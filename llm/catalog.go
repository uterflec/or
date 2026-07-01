package llm

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:generate go run ./internal/genmodels -output catalog.generated.json

// generatedCatalogJSON is checked into the repository and embedded so normal
// builds and application startup never depend on the network or working directory.
//
//go:embed catalog.generated.json
var generatedCatalogJSON []byte

// Startup path:
//
//	builtInModelRegistry = newBuiltInModelRegistry()
//	-> builtInModels()
//	-> json.Unmarshal(generatedCatalogJSON, &models)
//	-> each catalog entry is decoded as Model, so Model.UnmarshalJSON runs
//	-> newBuiltInModelRegistry registers the decoded models
var builtInModelRegistry = newBuiltInModelRegistry()

func newBuiltInModelRegistry() *ModelRegistry {
	registry := NewModelRegistry()
	for _, model := range builtInModels() {
		if err := registry.Register(model); err != nil {
			panic(err)
		}
	}
	return registry
}

func builtInModels() (models []Model) {
	err := json.Unmarshal(generatedCatalogJSON, &models)
	if err != nil {
		panic(fmt.Errorf("decode embedded model catalog: %w", err))
	}
	return
}
