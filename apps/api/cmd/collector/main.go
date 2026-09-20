package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/config"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/database"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/logger"
)

func main() {
	collectorFlag := flag.String("collector", "all", "Nome do coletor a executar (ex: 'santacasa', 'moinhos', 'saolucas', 'unimed', 'doctorclin', 'fleury', 'hcpa', 'divina', 'maededeus', ou 'all')")
	queryFlag := flag.String("query", "enfermagem", "Termo para busca de vagas (ex: 'enfermagem', 'técnico', vazio para todas)")
	limitFlag := flag.Int("limit", 10, "Quantidade máxima de vagas a listar no terminal por coletor (0 para todas)")
	jsonFlag := flag.Bool("json", false, "Exibir resultado em formato JSON")
	persistFlag := flag.Bool("persist", false, "Persistir vagas no banco de dados PostgreSQL")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	reg := collector.DefaultRegistry(20 * time.Second)
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

	runPersistence(ctx, targets, *queryFlag)
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
	}
}

func runPersistence(ctx context.Context, targets []collector.Collector, query string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Falha ao carregar configuração: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel, cfg.AppEnv)
	dbInstance, err := database.New(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Falha ao conectar ao banco de dados: %v\n", err)
		os.Exit(1)
	}
	defer dbInstance.Close()

	repo := database.NewJobRepository(dbInstance.Pool, log)
	svc := collector.NewService(repo, nil, nil, log)

	fmt.Printf("🚀 Executando pipeline completo de coleta e persistência (%d coletores)...\n", len(targets))

	var totalFound, totalInserted, totalUpdated, totalDuplicates, totalFailed int

	for _, c := range targets {
		fmt.Printf("\n▶️  Iniciando coleta: %s...\n", c.Name())
		res, err := svc.CollectFrom(ctx, c, collector.SearchQuery{Query: query})
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Erro durante a coleta de %s: %v\n", c.Name(), err)
			continue
		}

		totalFound += res.TotalFound
		totalInserted += res.Inserted
		totalUpdated += res.Updated
		totalDuplicates += res.Duplicates
		totalFailed += res.Failed

		fmt.Printf("   Fonte:       %s\n", res.CollectorName)
		fmt.Printf("   Encontradas: %d | Inseridas: %d | Atualizadas: %d | Duplicadas: %d | Falhas: %d (%v)\n",
			res.TotalFound, res.Inserted, res.Updated, res.Duplicates, res.Failed, res.Duration)
	}

	fmt.Printf("\n============================================================\n")
	fmt.Printf("📊 RESULTADO CONSOLIDADO DA INGESTÃO:\n")
	fmt.Printf("   Total Encontradas: %d\n", totalFound)
	fmt.Printf("   Total Inseridas:   %d\n", totalInserted)
	fmt.Printf("   Total Atualizadas: %d\n", totalUpdated)
	fmt.Printf("   Total Duplicadas:  %d\n", totalDuplicates)
	fmt.Printf("   Total Falhas:      %d\n", totalFailed)
	fmt.Printf("============================================================\n")
}
