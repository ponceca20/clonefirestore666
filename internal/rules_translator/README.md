# Módulo `rules_translator` - Documentación Completa

## 1. Propósito y Rol en el Proyecto

El módulo `rules_translator` es un subsistema especializado dentro del clon de Firestore que se encarga de **parsear, traducir, validar, optimizar y desplegar** las reglas de seguridad (`firestore.rules`) escritas en el lenguaje oficial de Firebase/Firestore, convirtiéndolas en un formato interno eficiente y compatible con el motor de seguridad de tu sistema.

Su objetivo es lograr **compatibilidad total** con la sintaxis y semántica de Firestore, permitiendo que los usuarios puedan importar y utilizar reglas existentes de Firebase sin modificaciones, y que el sistema pueda evaluarlas con alto rendimiento y seguridad.

## 2. Arquitectura y Componentes

El módulo está diseñado siguiendo principios de **arquitectura hexagonal (puertos y adaptadores)**, lo que permite desacoplar la lógica de parsing/traducción de la infraestructura y facilitar la integración, pruebas y extensibilidad.

### Componentes principales:

- **Dominio (`domain/`)**: Define los modelos, puertos (interfaces) y contratos para parsing, traducción, caché, optimización y despliegue de reglas.
- **Adaptadores (`adapter/`)**: Implementaciones concretas de los puertos, incluyendo:
  - Parser moderno (`parser/modern_parser.go`): Lexer y parser optimizados para la sintaxis de Firestore.
  - Traductor rápido (`usecase/fast_translator.go`): Traduce el AST a reglas internas.
  - Caché en memoria (`adapter/memory_cache.go`): Para acelerar traducciones repetidas.
  - Optimizador de reglas (`adapter/rules_optimizer.go`): Opcional, mejora rendimiento y elimina redundancias.
  - Deployer (`adapter/rules_deployer.go`): Despliega reglas al motor de seguridad.
- **CLI (`cmd/rules_importer/main.go`)**: Aplicación de línea de comandos para importar, validar y desplegar reglas desde archivos.
- **Tests (`test/`)**: Pruebas exhaustivas de compatibilidad, rendimiento y edge cases.

## 3. Flujo de Trabajo y Funcionalidad

### a) Parsing

- El parser convierte un archivo `.rules` en un AST (`FirestoreRuleset`), compatible con todas las características de Firestore (matches anidados, wildcards, operaciones, condiciones complejas, etc).

### b) Traducción

- El traductor convierte el AST en una lista de reglas internas (`SecurityRule`), mapeando operaciones, condiciones y rutas a un formato eficiente para evaluación en Go.

### c) Validación

- Se valida la sintaxis, semántica y consistencia de las reglas, detectando errores y advertencias antes de desplegar.

### d) Optimización (opcional)

- El optimizador puede consolidar reglas redundantes, simplificar condiciones y ajustar prioridades para mejorar el rendimiento.

### e) Caché

- Traducciones y resultados pueden ser cacheados para acelerar despliegues y pruebas repetidas.

### f) Despliegue

- El deployer valida y guarda las reglas en el motor de seguridad (MongoDB, memoria, etc), con soporte para rollback, historial y validación post-despliegue.

## 4. APIs y Contratos

El módulo expone **interfaces (puertos)** para integración:

- `RulesParser`: Parseo de archivos o strings a AST.
- `RulesTranslator`: Traducción de AST a reglas internas.
- `RulesCache`: Caché de traducciones.
- `RulesDeployer`: Despliegue seguro de reglas.
- `RulesOptimizer`: Optimización avanzada.

Ver `domain/ports.go` para detalles de métodos y estructuras.

## 5. CLI: Uso como Herramienta Independiente

El archivo `cmd/rules_importer/main.go` implementa una CLI para importar reglas:

```sh
go run cmd/rules_importer/main.go -rules=firestore.rules -project=myapp -database=default
```

Opciones:
- `-dry-run`: Solo valida y traduce, no despliega.
- `-validate-only`: Solo valida sintaxis.
- `-output=json`: Salida en JSON.
- `-optimize=false`: Desactiva optimización.

Esto permite probar reglas y despliegues fuera del ciclo de vida del servidor principal.

## 6. Integración con el Proyecto Principal

### a) Como Módulo Integrado

- **Recomendado**: Usar el paquete como librería Go, importando los puertos y adaptadores en el backend principal.
- Ejemplo de integración:
  ```go
  import (
      "firestore-clone/internal/rules_translator/domain"
      "firestore-clone/internal/rules_translator/adapter"
      "firestore-clone/internal/rules_translator/usecase"
  )

  // Inicializar componentes
  parser := adapter.NewModernParserInstance()
  optimizer := adapter.NewRulesOptimizer(nil)
  cache := adapter.NewMemoryCache(nil)
  translator := usecase.NewFastTranslator(cache, optimizer, nil)

  // Parsear y traducir reglas
  parseResult, err := parser.ParseString(ctx, rulesContent)
  translationResult, err := translator.Translate(ctx, parseResult.Ruleset)
  ```

- El resultado (`[]*SecurityRule`) puede ser pasado al motor de seguridad (`SecurityRulesEngine`) para evaluación en tiempo real.

### b) Como Microservicio Independiente

- **Opcional**: Ejecutar el CLI como microservicio, comunicándose vía archivos, REST o RPC.
- Útil para despliegues CI/CD, validación offline o integración con pipelines externos.
- El microservicio puede exponer endpoints para parsear, traducir y desplegar reglas.

### c) Recomendaciones

- Para máxima integración y rendimiento, **usar como módulo Go** dentro del backend principal.
- El CLI es útil para pruebas, validación y despliegue automatizado, pero no es obligatorio para producción.
- El motor de seguridad debe consumir las reglas traducidas y optimizadas para garantizar compatibilidad Firestore.

## 7. Casos de Uso

- **Importar reglas existentes de Firebase** sin cambios.
- **Validar reglas** antes de desplegar (CI/CD).
- **Optimizar reglas** para grandes sistemas multiempresa.
- **Desplegar reglas** de forma segura con rollback e historial.
- **Probar reglas** con tests exhaustivos de compatibilidad.

## 8. Extensibilidad

- Puedes implementar tus propios adaptadores para caché, optimización o despliegue.
- El parser y traductor pueden ser extendidos para soportar nuevas features de Firestore.

## 9. Referencias y Pruebas

- Ver `test/` para pruebas de compatibilidad, rendimiento y edge cases.
- Fixtures con reglas reales de Firestore en `test/fixtures/`.
- El módulo ha sido probado con reglas oficiales y casos complejos.

---

**Resumen**:  
El módulo `rules_translator` es el **puente entre las reglas Firestore originales y el motor de seguridad** de tu clon, asegurando compatibilidad, rendimiento y facilidad de integración. Puede usarse como librería Go o como microservicio, según las necesidades del proyecto.

