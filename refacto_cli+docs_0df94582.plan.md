---
name: Refacto CLI+Docs
overview: Refactoring large du CLI et des couches liées, correction d’anomalies, et mise à jour complète de la documentation/README, avec alignement sur le code (FFT 500 000 bits) et création du point d’entrée `cmd/fibcalc`.
todos:
  - id: audit-cli
    content: Auditer flux CLI/config/formatage et recenser anomalies
    status: pending
  - id: refactor-cli
    content: Refactoriser parsing, alias flags, sortie, erreurs, logging
    status: pending
  - id: docs-sync
    content: Aligner README/Docs (Go, FFT, dates, noms)
    status: pending
  - id: tests
    content: Mettre à jour/ajouter tests CLI + exécuter checks
    status: pending
isProject: false
---

# Plan de refactorisation CLI et documentation

## Portée et principes

- **Portée**: refonte large du CLI + modules internes connexes + docs/README (selon votre choix), en se basant sur la structure actuelle ([`internal/app`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/app), [`internal/config`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/config), [`internal/cli`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/cli), [`internal/errors`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/errors), [`internal/ui`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/ui)).
- **Source de vérité**: aligner la doc sur le code (FFT 500 000 bits).
- **Entrée CLI**: création du binaire principal `cmd/fibcalc` et vérification des liens avec le Makefile.

## Audit technique CLI (lecture)

- Cartographier le flux d’exécution: `New() -> ParseConfig() -> Run()` ([`internal/app/app.go`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/app/app.go)), routes vers `server/tui/repl`.
- Analyser la configuration: `ParseConfig()` et overrides env ([`internal/config/config.go`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/config/config.go), [`internal/config/env.go`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/config/env.go)), repérer duplications et validations.
- Examiner formatage/sortie CLI: `output.go`, `ui.go`, `presenter.go`, `repl.go` ([`internal/cli`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/cli)), notamment duplication de logique hex et rendu.
- Vérifier gestion d’erreurs et couleurs: [`internal/errors/handler.go`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/errors/handler.go), [`internal/ui`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/ui).

## Refactoring CLI (implémentation)

- **Créer le point d’entrée principal**: `cmd/fibcalc/main.go` + wiring vers `internal/app`.
- **Refactoriser `ParseConfig()`**: séparer en sous-fonctions (définition des flags, parsing, env overrides, validation) pour améliorer lisibilité et testabilité.
- **Corriger les alias de flags**: éviter les doubles définitions; garantir que `-d/-o/-q/-c` fonctionnent correctement.
- **Centraliser le formatage de sortie**: supprimer duplications (hex/resultats) entre `app.go` et `output.go`; unifier `DisplayResult*` et `Format*`.
- **Standardiser gestion d’erreurs**: remplacer usages directs de `ui.Color*()` dans le REPL par `apperrors.HandleCalculationError()` + `ColorProvider`.
- **Intégrer logging structuré** côté CLI quand pertinent (erreurs, événements), sans polluer la sortie utilisateur.

## Documentation complète (audit + corrections)

- **Aligner Go version**: corriger incohérences entre [`README.md`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/README.md) et [`Docs/PERFORMANCE.md`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/Docs/PERFORMANCE.md).
- **Aligner seuil FFT** (500 000 bits): mettre à jour [`Docs/algorithms/FFT.md`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/Docs/algorithms/FFT.md), [`Docs/PERFORMANCE.md`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/Docs/PERFORMANCE.md), [`README.md`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/README.md).
- **Dates/labels**: harmoniser la date de mise à jour et la casse “fibcalc” vs “FibCalc” sur l’ensemble des docs.
- **CLI usage**: vérifier que l’aide, les flags, et la section configuration du README reflètent le comportement réel (et les env vars).

## Tests et validation

- Ajouter/ajuster tests unitaires ciblés sur parsing CLI et alias de flags (dans [`internal/config`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/config)), plus tests de sortie si nécessaire ([`internal/cli`](C:/Users/agbru/OneDrive/Documents/GitHub/FibGo/internal/cli)).
- Exécuter `make test-short` puis `make lint` si possible; compléter par tests spécifiques CLI si existants.

## Livrables

- Refactor CLI complet + point d’entrée principal.
- Docs et README alignés sur le code.
- Tests mis à jour pour garantir stabilité CLI.

## Risques / points d’attention

- Changement d’API CLI peut impacter scripts utilisateurs; documenter les modifications si comportement visible.
- Les seuils FFT étant alignés sur le code, surveiller toute régression de performance liée à la doc seule.