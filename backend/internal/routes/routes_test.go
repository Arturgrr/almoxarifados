package routes

import (
	"net/http"
	"strings"
	"testing"

	"almoxarifados-a2g/internal/docs"
)

func TestTodasAsRotasEstaoDocumentadas(t *testing.T) {
	app, err := New(nil, Config{FrontendURL: "http://localhost:5173"})
	if err != nil {
		t.Fatal(err)
	}

	documentadas := map[string]bool{}
	for _, op := range docs.Operacoes() {
		documentadas[op.Metodo+" "+op.Caminho] = true
	}

	registradas := map[string]bool{}
	for _, r := range app.GetRoutes(true) {
		if r.Method == http.MethodHead || !strings.HasPrefix(r.Path, "/api/") {
			continue
		}
		rota := r.Method + " " + r.Path
		registradas[rota] = true
		if !documentadas[rota] {
			t.Errorf("rota sem documentação OpenAPI: %s (adicione em internal/docs/operacoes.go)", rota)
		}
	}
	for rota := range documentadas {
		if !registradas[rota] {
			t.Errorf("rota documentada que não existe: %s", rota)
		}
	}
}
