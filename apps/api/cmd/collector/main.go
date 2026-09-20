package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/config"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/database"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/logger"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/scheduler"
)

func main() {
	collectorFlag := flag.String("collector", "all", "Nome do coletor a executar (ex: 'santacasa', 'moinhos', 'saolucas', 'unimed', 'doctorclin', 'fleury', 'hcpa', 'divina', 'maededeus', ou 'all')")
	queryFlag := flag.String("query", "enfermagem", "Termo para busca de vagas (ex: 'enfermagem', 'técnico', vazio para todas)")
	limitFlag := flag.Int("limit", 10, "Quantidade máxima de vagas a listar no terminal por coletor (0 para todas)")
	jsonFlag := flag.Bool("json", false, "Exibir resultado em formato JSON")
	persistFlag := flag.Bool("persist", false, "Persistir vagas no banco de dados PostgreSQL")
	concurrencyFlag := flag.Int("concurrency", 0, "Quantidade máxima de coletores simultâneos (0 utiliza configuração padrão)")
	scheduleFlag := flag.Bool("schedule", false, "Executar em modo agendador contínuo com cron")
	cronScheduleFlag := flag.String("cron", "", "Expressão cron customizada para agendador (ex: '0 */2 * * *')")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Falha ao carregar configuração: %v\n", err)
		os.Exit(1)
	}

	concurrency := cfg.CollectorConcurrency
	if *concurrencyFlag > 0 {
		concurrency = *concurrencyFlag
	}

	cronExpr := cfg.CollectorCronSchedule
	if *cronScheduleFlag != "" {
		cronExpr = *cronScheduleFlag
	}

	log := logger.New(cfg.LogLevel, cfg.AppEnv)

	httpClient := collector.NewResilientHTTPClient(collector.ResilientClientConfig{
		Timeout:           20 * time.Second,
		MaxRetries:        3,
		InitialRetryDelay: 200 * time.Millisecond,
		DefaultRPS:        cfg.CollectorRateLimitRPS,
		DefaultBurst:      cfg.CollectorRateLimitBurst,
	}, log)

	reg := collector.NewDefaultRegistry(httpClient, 20*time.Second)

	if *scheduleFlag {
		runSchedulerMode(cfg, reg, log, cronExpr, concurrency, *queryFlag)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var targets []collector.Collector
	if strings.ToLower(*collectorFlag) == "all" {
		targets = reg.All()
	} else {
		c, ok := reg.Get(strings.ToLower(*collectorFlag))
		if !ok {
			fmt.Fprintf(os.Stderr, "❌ Coletor %q não encontrado. Disponíveis: %s\n", *collectorFlag, strings.Join(reg.Names(), ", "))
			os.Exit(1)
		}
		targets = append(targets, c)
	}

	if !*persistFlag {
		runDryRun(ctx, targets, *queryFlag, *limitFlag, *jsonFlag)
		return
	}

	runPersistence(ctx, cfg, log, targets, *queryFlag, concurrency)
}

func runSchedulerMode(cfg *config.Config, reg *collector.Registry, log *slog.Logger, cronExpr string, concurrency int, query string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbInstance, err := database.New(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Falha ao conectar ao banco de dados: %v\n", err)
		os.Exit(1)
	}
	defer dbInstance.Close()

	repo := database.NewJobRepository(dbInstance.Pool, log)
	svc := collector.NewService(repo, nil, nil, log)

	unknownThreshold := time.Duration(cfg.CollectorStatusUnknownHours) * time.Hour
	expiredThreshold := time.Duration(cfg.CollectorStatusExpiredHours) * time.Hour

	sched, err := scheduler.NewScheduler(scheduler.Config{
		CronSchedule:     cronExpr,
		Concurrency:      concurrency,
		UnknownThreshold: unknownThreshold,
		ExpiredThreshold: expiredThreshold,
		Query:            collector.SearchQuery{Query: query},
	}, svc, reg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Falha ao inicializar scheduler: %v\n", err)
		os.Exit(1)
	}

	if err := sched.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Falha ao iniciar scheduler: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("⏰ Scheduler iniciado com expressão cron %q (concorrência: %d). Pressione Ctrl+C para encerrar.\n", cronExpr, concurrency)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Encerrando scheduler graciosamente...")
	sched.Stop()
	fmt.Println("✅ Scheduler finalizado.")
}

func runDryRun(ctx context.Context, targets []collector.Collector, query string, limit int, asJSON bool) {
	for _, c := range targets {
		if !asJSON {
			fmt.Printf("🔍 [DRY-RUN] Executando coleta no portal %q (filtro: %q)...\n\n", c.Name(), query)
		}

		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: query})
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Falha ao coletar vagas de %s: %v\n", c.Name(), err)
			continue
		}

		if len(jobs) == 0 {
			if !asJSON {
				fmt.Printf("ℹ️  Nenhuma vaga encontrada para %s com os critérios informados.\n\n", c.Name())
			}
			continue
		}

		count := len(jobs)
		if limit > 0 && limit < count {
			count = limit
		}

		if asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(jobs[:count])
			continue
		}

		fmt.Printf("✅ [%s] Encontradas %d vagas (exibindo as primeiras %d):\n\n", c.Name(), len(jobs), count)
		for i, j := range jobs[:count] {
			fmt.Printf("------------------------------------------------------------\n")
			fmt.Printf("📍 [%02d] %s\n", i+1, j.Title)
			fmt.Printf("   Instituição: %s\n", j.Company)
			fmt.Printf("   Local:       %s - %s\n", j.City, j.State)
			fmt.Printf("   Modalidade:  %s | Vínculo: %s\n", j.WorkMode, j.EmploymentType)
			fmt.Printf("   Link:        %s\n", j.SourceURL)
		}
		fmt.Printf("------------------------------------------------------------\n\n")
	}

	if !asJSON {
		fmt.Printf("💡 Dica: Para salvar no banco use: go run ./cmd/collector -collector=%s -persist\n", targets[0].Name())
		fmt.Printf("💡 Para ver em JSON use: go run ./cmd/collector -json\n")
		fmt.Printf("💡 Para rodar em modo scheduler use: go run ./cmd/collector -schedule\n")
	}
}

func runPersistence(ctx context.Context, cfg *config.Config, log *slog.Logger, targets []collector.Collector, query string, concurrency int) {
	dbInstance, err := database.New(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Falha ao conectar ao banco de dados: %v\n", err)
		os.Exit(1)
	}
	defer dbInstance.Close()

	repo := database.NewJobRepository(dbInstance.Pool, log)
	svc := collector.NewService(repo, nil, nil, log)

	fmt.Printf("🚀 Executando pipeline completo de coleta e persistência (%d coletores, concorrência: %d)...\n\n", len(targets), concurrency)

	metrics, err := svc.CollectAll(ctx, targets, collector.SearchQuery{Query: query}, concurrency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Falha durante a coleta concorrente: %v\n", err)
	}

	for _, name := range getSortedKeys(metrics.ByCollector) {
		cm := metrics.ByCollector[name]
		statusIcon := "✅"
		if cm.Error != "" {
			statusIcon = "⚠️"
		}
		fmt.Printf("%s Fonte: %-15s | Encontradas: %-3d | Inseridas: %-3d | Atualizadas: %-3d | Duplicadas: %-3d | Falhas: %-2d (%v)\n",
			statusIcon, cm.CollectorName, cm.TotalFound, cm.Inserted, cm.Updated, cm.Duplicates, cm.Failed, cm.Duration)
		if cm.Error != "" {
			fmt.Printf("   ↳ Erro: %s\n", cm.Error)
		}
	}

	// Reconciliação de status
	unknownThreshold := time.Duration(cfg.CollectorStatusUnknownHours) * time.Hour
	expiredThreshold := time.Duration(cfg.CollectorStatusExpiredHours) * time.Hour
	reconcileRes, recErr := svc.ReconcileJobStatuses(ctx, unknownThreshold, expiredThreshold)
	if recErr != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Falha na reconciliação de status: %v\n", recErr)
	} else {
		fmt.Printf("\n🔄 RECONCILIAÇÃO DE STATUS:\n")
		fmt.Printf("   Vagas marcadas como UNKNOWN (> %dh sem serem vistas): %d\n", cfg.CollectorStatusUnknownHours, reconcileRes.MarkedUnknown)
		fmt.Printf("   Vagas marcadas como EXPIRED (> %dh sem serem vistas): %d\n", cfg.CollectorStatusExpiredHours, reconcileRes.MarkedExpired)
	}

	fmt.Printf("\n============================================================\n")
	fmt.Printf("📊 RESULTADO CONSOLIDADO DA INGESTÃO (Run ID: %s):\n", metrics.RunID)
	fmt.Printf("   Total Encontradas: %d\n", metrics.TotalFound)
	fmt.Printf("   Total Inseridas:   %d\n", metrics.TotalInserted)
	fmt.Printf("   Total Atualizadas: %d\n", metrics.TotalUpdated)
	fmt.Printf("   Total Duplicadas:  %d\n", metrics.TotalDuplicates)
	fmt.Printf("   Total Falhas:      %d\n", metrics.TotalFailed)
	fmt.Printf("   Duração Total:     %v\n", metrics.Duration)
	fmt.Printf("============================================================\n")
}

func getSortedKeys(m map[string]collector.CollectorMetrics) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
