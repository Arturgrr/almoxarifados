package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"almoxarifados-a2g/internal/docs"
)

func main() {
	saida := flag.String("saida", "../docs", "pasta onde os arquivos serão gravados")
	servidor := flag.String("servidor", "http://localhost:3000", "URL base da API usada no Swagger")
	flag.Parse()

	spec, err := docs.Gerar(*servidor)
	if err != nil {
		log.Fatal(err)
	}
	specJSON, err := docs.JSON(spec)
	if err != nil {
		log.Fatal(err)
	}
	specYAML, err := spec.MarshalYAML()
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(*saida, 0o755); err != nil {
		log.Fatal(err)
	}
	arquivos := map[string][]byte{
		"openapi.yaml": specYAML,
		"openapi.json": specJSON,
		"swagger.html": docs.PaginaSwagger(specJSON),
	}
	for nome, conteudo := range arquivos {
		caminho := filepath.Join(*saida, nome)
		if err := os.WriteFile(caminho, conteudo, 0o644); err != nil {
			log.Fatal(err)
		}
		log.Printf("gerado %s", caminho)
	}
}
