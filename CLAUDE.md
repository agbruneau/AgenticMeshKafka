# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **French technical monograph** (not a code project) titled "Interopérabilité en Écosystème d'Entreprise : Convergence des Architectures d'Intégration". It documents enterprise integration architecture patterns across three domains: Applications, Data, and Events.

**Central Thesis**: Interoperability is not binary but a continuum requiring a hybrid strategy (App → Data → Event) — from tight coupling to maximum decoupling, culminating in the "Entreprise Agentique".

## Structure

- **11 chapters** in `Chapitres/` (numbered 01-11, each in its own subdirectory)
- **Annexes** in `Annexes/Annexes` (Glossary, Technology Comparisons, Maturity Checklist, Bibliography)
- **TOC.md**: Detailed table of contents
- **INSTRUCTION.MD**: Complete editorial guidelines (MUST READ before editing)

## Editorial Guidelines

### Language and Voice

- **Quebec French professional voice** — use recognized Quebec terms (infonuagique, courriel)
- Avoid untranslated Anglo-Saxon jargon (except recognized technical terms)
- Expert tone without unnecessary jargon; vulgarize complex concepts

### Chapter Format

- Target length: ~8,000 words per chapter
- Structure: Introduction (10-15%), Development (75-80%), Conclusion/Transition (10%)
- Heading hierarchy: `##` for main sections, `###` for subsections, `####` sparingly
- Prefer **fluid prose over bullet lists**
- Each chapter ends with a structured **Résumé**
- Cross-references format: "Comme établi au chapitre II..." or "Le patron CDC présenté au chapitre IV..."

### Terminology

Always use the consolidated terminology from INSTRUCTION.MD, including:

- **Les Trois Domaines**: Applications (Le Verbe), Données (Le Nom), Événements (Le Signal)
- **23 architecture patterns** documented across chapters III-V
- First occurrence of acronyms: full form with acronym in parentheses

### Citations

- Prioritize 2023-2026 sources
- Reference industry leaders: Confluent, Apache, Google Cloud, Anthropic, Microsoft
- Format: "selon [Organisation, Année]" or Author (Year)

## The Three Integration Domains

| Domain | Metaphor | Focus | Chapter |
|--------|----------|-------|---------|
| Applications | Le Verbe | Orchestration, synchronous interactions | III |
| Données | Le Nom | State consistency, data accessibility | IV |
| Événements | Le Signal | Reactivity, maximum temporal decoupling | V |

## Narrative Flow

```
I. Problem → II. Theory → III-V. Solutions (App→Data→Event)
→ VI-VII. Foundations (Standards + Resilience)
→ VIII. Evolution → IX. Synthesis → X. Case Study → XI. Vision (Entreprise Agentique)
```
