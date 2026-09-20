package job

import (
	"errors"
	"strings"
)

var (
	// ErrNotNursingJob indica que a vaga não pertence ao domínio de enfermagem.
	ErrNotNursingJob = errors.New("vaga não pertence ao domínio de enfermagem")

	// ErrOutOfScopeLocation indica que a localidade da vaga não está dentro da Região Metropolitana de Porto Alegre/RS.
	ErrOutOfScopeLocation = errors.New("vaga fora do escopo territorial atendido")
)

// Blacklist de termos de exclusão estrita no título da vaga.
// Se qualquer um desses termos estiver presente no título, a vaga é descartada de imediato.
var nonNursingTitleKeywords = []string{
	"arquiteto", "arquiteta", "projetos executivos",
	"jovem aprendiz", "aprendiz",
	"hidraulica", "eletricista", "manutencao predial", "pedreiro", "pintor",
	"atendente de museu", "museu",
	"advogado", "advogada", "juridico", "contencioso",
	"engenheiro", "engenheira", "engenharia",
	"cozinheiro", "cozinheira", "auxiliar de cozinha", "nutricionista", "gastronomia",
	"analista de sistemas", "analista de ti", "desenvolvedor", "programador", "suporte de ti", "desenvolvimento de software",
	"professor", "professora", "docente", "pedagogo", "pedagoga", "colegio", "escola", "secretaria escolar", "secretario escolar",
	"porteiro", "vigia", "vigilante", "motorista",
	"telefonista",
	"medico", "medica", "residencia medica",
	"fisioterapeuta", "psicologo", "psicologa", "fonoaudiologo", "fonoaudiologa", "assistente social",
}

// Whitelist de termos positivos de enfermagem no título.
var nursingTitleKeywords = []string{
	"enferm",      // cobre: enfermagem, enfermeiro, enfermeira, enfermeir@
	"instrumenta", // cobre: instrumentador, instrumentadora, instrumentacao cirurgica
	"flebotom",    // cobre: flebotomista, flebotomia
	"vacinad",     // cobre: vacinador, vacinadora
}

// Termos específicos hospitalares de atuação típica de enfermagem.
var nursingSpecialtyKeywords = []string{
	"cme",
	"centro de material",
	"bloco cirurgico",
	"centro cirurgico",
	"hemodialise",
	"quimioterapia",
	"obstetra",
	"obstetricia",
	"triagem",
	"acolhimento",
	"imunizacao",
}

// Cidades da Região Metropolitana de Porto Alegre atendidas pelo projeto.
var metropolitanCitiesRS = map[string]struct{}{
	"porto alegre":    {},
	"canoas":          {},
	"novo hamburgo":   {},
	"sao leopoldo":    {},
	"gravatai":        {},
	"viamao":          {},
	"alvorada":        {},
	"cachoeirinha":    {},
	"esteio":          {},
	"sapucaia do sul": {},
	"guaiba":          {},
	"eldorado do sul": {},
	"campo bom":       {},
	"sapiranga":       {},
	"dois irmaos":     {},
	"estancia velha":  {},
	"taquara":         {},
	"parobe":          {},
	"montenegro":      {},
	"triunfo":         {},
	"charqueadas":     {},
}

// IsNursingJob valida se uma vaga pertence ao domínio de enfermagem com base no título e na descrição.
func IsNursingJob(title, description string) bool {
	normTitle := NormalizeText(title)
	if normTitle == "" {
		return false
	}

	// 1. Verificação de exclusão estrita no título (Blacklist)
	for _, term := range nonNursingTitleKeywords {
		if strings.Contains(normTitle, term) {
			return false
		}
	}

	// 2. Verificação de termos positivos no título (Whitelist)
	for _, term := range nursingTitleKeywords {
		if strings.Contains(normTitle, term) {
			return true
		}
	}

	// 3. Verificação de especialidades com contexto de enfermagem ou UTI
	for _, term := range nursingSpecialtyKeywords {
		if strings.Contains(normTitle, term) {
			return true
		}
	}

	// 4. Verificação de UTI: se contém "uti" no título
	if strings.Contains(normTitle, "uti") {
		return true
	}

	// 5. Fallback para a descrição: busca por menção obrigatória a COREN ou formação em Enfermagem
	normDesc := NormalizeText(description)
	if normDesc != "" {
		if strings.Contains(normDesc, "coren") ||
			strings.Contains(normDesc, "tecnico em enfermagem") ||
			strings.Contains(normDesc, "tecnico de enfermagem") ||
			strings.Contains(normDesc, "auxiliar de enfermagem") ||
			strings.Contains(normDesc, "graduacao em enfermagem") {
			return true
		}
	}

	return false
}

// IsTargetLocation valida se a localidade da vaga pertence à Região Metropolitana de Porto Alegre / RS.
func IsTargetLocation(city, state string) bool {
	normState := strings.ToUpper(NormalizeText(state))
	if normState != "RS" && normState != "RIO GRANDE DO SUL" {
		return false
	}

	normCity := NormalizeText(city)
	if normCity == "" {
		return false
	}

	_, ok := metropolitanCitiesRS[normCity]
	return ok
}
