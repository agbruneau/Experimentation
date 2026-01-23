# Plan: Implémentation TUI avec Bubbletea pour Fibonacci Calculator

## Analyse d'Opportunité

### Avantages

| Aspect | Bénéfice |
|--------|----------|
| **Architecture prête** | Les interfaces `ProgressReporter` et `ResultPresenter` découplent déjà la présentation - aucune modification du code métier nécessaire |
| **UX améliorée** | Navigation intuitive, visualisation temps réel, interactions riches vs CLI brute |
| **Écosystème Charm** | Bubbles (composants), Lipgloss (styles), documentation excellente, 10k+ projets en production |
| **Cohérence thèmes** | Intégration naturelle avec `internal/ui/themes.go` existant |
| **Mode interactif supérieur** | Remplace le REPL textuel par une interface navigable |

### Considérations

| Aspect | Impact |
|--------|--------|
| **Nouvelles dépendances** | +3 packages Charm (~2MB binaire) |
| **Complexité** | Pattern Elm Architecture à maîtriser |
| **Maintenance** | Nouveau code à tester et maintenir |
| **Compatibilité** | Nécessite terminal supportant ANSI (couvert par 99% des terminaux modernes) |

### Verdict: **Recommandé**

L'architecture existante est idéalement préparée pour cette extension. Le coût d'implémentation est modéré grâce aux interfaces déjà en place.

---

## Approche d'Intégration

### Principe: Nouvelle implémentation des interfaces existantes

```
                    ┌─────────────────────────┐
                    │   Orchestration Layer   │
                    │  (aucune modification)  │
                    └───────────┬─────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              │                 │                 │
              ▼                 ▼                 ▼
    ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
    │ CLIProgressRep. │ │ TUIProgressRep. │ │ NullProgressRep │
    │ CLIResultPres.  │ │ TUIResultPres.  │ │                 │
    └─────────────────┘ └─────────────────┘ └─────────────────┘
           CLI                 TUI              Quiet/Tests
```

---

## Structure du Package

```
internal/tui/
├── tui.go              # Point d'entrée, tea.NewProgram()
├── model.go            # Modèle racine Bubbletea
├── messages.go         # Types de messages (ProgressMsg, ResultMsg, etc.)
├── commands.go         # tea.Cmd pour opérations async
├── keys.go             # Bindings clavier
├── styles.go           # Styles Lipgloss intégrés avec internal/ui
├── presenter.go        # TUIProgressReporter, TUIResultPresenter
│
├── views/
│   ├── home.go         # Écran d'accueil
│   ├── calculator.go   # Saisie N + sélection algorithme
│   ├── progress.go     # Barre de progression temps réel
│   ├── results.go      # Affichage résultat
│   ├── comparison.go   # Comparaison multi-algorithmes
│   └── help.go         # Aide et raccourcis
│
└── components/
    ├── input.go        # Champ de saisie numérique
    ├── selector.go     # Liste de sélection algorithme
    ├── progressbar.go  # Barre de progression
    └── statusbar.go    # Barre d'état en bas
```

---

## Fichiers à Modifier

| Fichier | Modification |
|---------|--------------|
| `internal/config/config.go` | Ajouter `TUIMode bool` et flag `--tui` |
| `internal/app/app.go` | Ajouter `runTUI()` et dispatch dans `Run()` |
| `go.mod` | Ajouter dépendances Charm |

---

## Dépendances à Ajouter

```go
require (
    github.com/charmbracelet/bubbletea v1.3.5
    github.com/charmbracelet/bubbles v0.20.0
    github.com/charmbracelet/lipgloss v1.1.0
)
```

---

## Pont Bubbletea ↔ Orchestration

Le défi principal: convertir le channel `fibonacci.ProgressUpdate` en messages Bubbletea.

```go
// messages.go
type ProgressMsg struct {
    Update fibonacci.ProgressUpdate
}

// commands.go - Écoute le channel et émet des messages
func listenForProgress(ch <-chan fibonacci.ProgressUpdate) tea.Cmd {
    return func() tea.Msg {
        update, ok := <-ch
        if !ok {
            return ProgressDoneMsg{}
        }
        return ProgressMsg{Update: update}
    }
}

// presenter.go
type TUIProgressReporter struct {
    program *tea.Program  // Référence au programme Bubbletea
}

func (t *TUIProgressReporter) DisplayProgress(wg *sync.WaitGroup,
    progressChan <-chan fibonacci.ProgressUpdate, numCalculators int, _ io.Writer) {
    defer wg.Done()
    for update := range progressChan {
        t.program.Send(ProgressMsg{Update: update})
    }
}
```

---

## Design UI/UX

### Écran Principal
```
╭─────────────────────────────────────────────────────────╮
│  Fibonacci Calculator                     Theme: Dark   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│   [Contenu dynamique selon la vue]                      │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  Tab:Navigate  Enter:Select  Esc:Back  ?:Help  q:Quit   │
╰─────────────────────────────────────────────────────────╯
```

### Raccourcis Clavier

| Touche | Action |
|--------|--------|
| `Tab` / `Shift+Tab` | Navigation entre champs |
| `Enter` | Confirmer / Démarrer calcul |
| `Esc` | Annuler / Retour |
| `q` / `Ctrl+C` | Quitter |
| `?` / `F1` | Aide |
| `c` | Nouvelle calculation |
| `m` | Mode comparaison |
| `t` | Changer thème |

---

## Phases d'Implémentation

### Phase 1: Fondation
- Ajouter dépendances Charm
- Créer structure `internal/tui/`
- Modèle racine avec navigation entre états
- Styles Lipgloss intégrés avec themes existants
- Flag `--tui` et dispatch dans app.go
- Vue Home basique

### Phase 2: Calcul Simple
- Vue Calculator (saisie N, sélection algo)
- Vue Progress (barre de progression, ETA)
- Pont TUIProgressReporter
- Vue Results (affichage résultat)
- Annulation avec Esc

### Phase 3: Comparaison
- Vue Comparison (multi-progress)
- TUIResultPresenter complet
- Table de résultats
- Vérification de cohérence

### Phase 4: Polish
- Vue Settings (thème, defaults)
- Actions résultat (sauvegarder, hex, copier)
- Responsive layout
- Tests complets

### Phase 5: Documentation
- Mise à jour du README.md avec section TUI (installation, usage, captures d'écran)
- Documentation des raccourcis clavier dans le README
- Mise à jour de CLAUDE.md avec les nouveaux packages et conventions TUI
- Documentation API interne du package `internal/tui/`
- Mise à jour des exemples d'utilisation CLI vs TUI
- Ajout de la documentation des dépendances Charm dans go.mod comments
- Vérification et mise à jour de tous les fichiers .md existants pour cohérence

---

## Vérification

1. `go build ./...` - Compilation sans erreur
2. `fibcalc --tui` - Lancement TUI
3. Navigation entre vues avec Tab/Enter/Esc
4. Calcul F(1000) avec progression visible
5. Comparaison tous algorithmes
6. `make test` - Tests passent
7. `make lint` - Pas de violations

---

## Fichiers Critiques à Référencer

- [interfaces.go](internal/orchestration/interfaces.go) - Interfaces ProgressReporter/ResultPresenter
- [config.go](internal/config/config.go) - Pattern pour ajouter flag TUI
- [app.go](internal/app/app.go) - Point d'intégration runTUI()
- [themes.go](internal/ui/themes.go) - Thèmes à intégrer avec Lipgloss
- [progress_eta.go](internal/cli/progress_eta.go) - Logique ETA réutilisable
