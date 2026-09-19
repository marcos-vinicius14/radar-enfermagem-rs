package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/config"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/database"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/logger"
)

func main() {
	queryFlag := flag.String("query", "enfermagem", "Termo para busca de vagas (ex: 'enfermagem', 'técnico', vazio para todas)")
	limitFlag := flag.Int("limit", 10, "Quantidade máxima de vagas a listar no terminal (0 para todas)")
	jsonFlag := flag.Bool("json", false, "Exibir resultado em formato JSON")
	persistFlag := flag.Bool("persist", false, "Persistir vagas no banco de dados PostgreSQL")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	scCollector := collector.NewSantaCasaCollector(nil, 15*time.Second)

	if !*persistFlag {
		if !*jsonFlag {
			fmt.Printf("🔍 [DRY-RUN] Executando coleta ao vivo no portal da Santa Casa (filtro: %q)...\n\n", *queryFlag)
		}
		jobs, err := scCollector.Collect(ctx, collector.SearchQuery{Query: *queryFlag})
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Falha ao coletar vagas: %v\n", err)
			os.Exit(1)
		}

		if len(jobs) == 0 {
			fmt.Println("ℹ️  Nenhuma vaga encontrada com os critérios informados.")
			return
		}

		limit := len(jobs)
		if *limitFlag > 0 && *limitFlag < limit {
			limit = *limitFlag
		}

		if *jsonFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(jobs[:limit])
			return
		}

		fmt.Printf("✅ Encontradas %d vagas (exibindo as primeiras %d):\n\n", len(jobs), limit)
		for i, j := range jobs[:limit] {
			fmt.Printf("------------------------------------------------------------\n")
			fmt.Printf("📍 [%02d] %s\n", i+1, j.Title)
			fmt.Printf("   Instituição: %s\n", j.Company)
			fmt.Printf("   Local:       %s - %s\n", j.City, j.State)
			fmt.Printf("   Modalidade:  %s | Vínculo: %s\n", j.WorkMode, j.EmploymentType)
			fmt.Printf("   Link:        %s\n", j.SourceURL)
		}
		fmt.Printf("------------------------------------------------------------\n")
		fmt.Printf("\n💡 Dica: Para salvar no banco use: go run ./cmd/collector -persist\n")
		fmt.Printf("💡 Para ver em JSON use: go run ./cmd/collector -json\n")
		return
	}

	// Modo com persistência no banco
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

	fmt.Printf("🚀 Executando pipeline completo de coleta e persistência...\n")
	res, err := svc.CollectFrom(ctx, scCollector, collector.SearchQuery{Query: *queryFlag})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Erro durante o pipeline de coleta: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n📊 Resultado da Ingestão:\n")
	fmt.Printf("   Fonte:       %s\n", res.CollectorName)
	fmt.Printf("   Encontradas: %d\n", res.TotalFound)
	fmt.Printf("   Inseridas:   %d\n", res.Inserted)
	fmt.Printf("   Atualizadas: %d\n", res.Updated)
	fmt.Printf("   Duplicadas:  %d\n", res.Duplicates)
	fmt.Printf("   Falhas:      %d\n", res.Failed)
	fmt.Printf("   Duração:     %v\n", res.Duration)
}
