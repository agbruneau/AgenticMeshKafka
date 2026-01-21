# 15.4 Architecture Decision Records (ADR)

## Résumé

Les **Architecture Decision Records (ADR)** sont des documents courts qui capturent les décisions d'architecture importantes avec leur contexte et leurs conséquences. Ils constituent la mémoire de votre architecture et évitent de répéter les erreurs passées.

## Points clés

- Un ADR par décision significative
- Court et focalisé (1-2 pages max)
- Contexte et justification essentiels
- Les ADR ne changent pas - on crée un nouvel ADR pour les superseder

---

## Pourquoi Documenter les Décisions ?

```
┌─────────────────────────────────────────────────────────────────┐
│              SANS ADR                    AVEC ADR               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  6 mois plus tard...                  6 mois plus tard...      │
│                                                                 │
│  "Pourquoi on utilise Kafka ?"        → Voir ADR-007           │
│  "Je sais pas, c'était avant moi"     "Choisi pour X, Y, Z"    │
│                                        "Alternatives: A, B"    │
│  "Bon, on migre vers RabbitMQ"        "Trade-offs acceptés"    │
│  └── 3 mois de travail                                         │
│      pour redécouvrir pourquoi        → Évite de refaire       │
│      Kafka avait été choisi             les mêmes erreurs      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Structure d'un ADR

### Template Standard

```markdown
# ADR-[NUMERO]: [TITRE COURT]

## Statut
[Proposé | Accepté | Déprécié | Remplacé par ADR-XXX]

## Contexte
Quel est le problème ou la situation qui nécessite une décision ?
(Description factuelle, pas d'opinion)

## Décision
Quelle est la décision prise ?
(Affirmation claire et concise)

## Conséquences
Quelles sont les implications positives et négatives ?
(Liste des impacts, trade-offs acceptés)

## Alternatives Considérées
Quelles autres options ont été évaluées et pourquoi rejetées ?
(Montre que la décision est réfléchie)

---
Date: YYYY-MM-DD
Auteur(s): Nom(s)
```

---

## Exemple Complet : Écosystème Assurance

### ADR-001: Utilisation d'une API Gateway centralisée

```markdown
# ADR-001: API Gateway Centralisée pour l'Exposition des Services

## Statut
Accepté

## Contexte
Notre écosystème assurance expose 8 services mock (Quote Engine, Policy Admin,
Claims, Billing, Customer Hub, Document Mgmt, Notifications, External Rating).

Chaque service a sa propre API, avec des besoins communs :
- Authentification (API Key, JWT)
- Rate limiting (protection contre les abus)
- Logging centralisé
- Transformation de réponses

Les partenaires externes (courtiers, comparateurs) ont besoin d'un point
d'entrée unique et stable.

## Décision
Nous implémentons une API Gateway centralisée qui :
- Expose un point d'entrée unique : /gateway/*
- Route les requêtes vers les services backend
- Applique l'authentification uniformément
- Implémente le rate limiting par client
- Centralise les logs d'accès

## Conséquences

### Positives
- ✅ Contrat d'API stable pour les partenaires
- ✅ Sécurité appliquée uniformément
- ✅ Monitoring centralisé
- ✅ Évolution des backends transparente

### Négatives (trade-offs acceptés)
- ❌ Point de défaillance unique (mitigé par redundance)
- ❌ Latence additionnelle (~5-10ms)
- ❌ Complexité de configuration des routes

## Alternatives Considérées

1. **Exposition directe des services**
   - Rejeté : Pas de sécurité uniforme, URLs instables

2. **Service Mesh (Istio/Linkerd)**
   - Rejeté : Overkill pour notre taille, complexité opérationnelle

3. **BFF par partenaire**
   - Partiellement retenu : BFF Mobile et BFF Broker en plus de la Gateway

---
Date: 2024-01-10
Auteur: Équipe Architecture
```

---

### ADR-007: Event Sourcing pour le Cycle de Vie des Polices

```markdown
# ADR-007: Event Sourcing pour l'Historique des Polices

## Statut
Accepté

## Contexte
Le système Policy Admin doit :
- Maintenir un historique complet de toutes les modifications
- Permettre l'audit réglementaire (qui a fait quoi, quand)
- Supporter la reconstruction d'un état passé
- Répondre aux exigences de conformité RGPD (traçabilité)

L'approche CRUD traditionnelle perd l'historique des modifications
intermédiaires.

## Décision
Nous implémentons Event Sourcing pour les entités Policy :
- Chaque modification est un événement immuable (append-only)
- L'état actuel est reconstruit par replay des événements
- Un Event Store dédié stocke les événements
- Des projections optimisées pour les requêtes courantes

Événements définis :
- PolicyDrafted
- PolicyActivated
- PolicyAmended
- PolicyCancelled
- PolicyRenewed

## Conséquences

### Positives
- ✅ Audit trail complet et immuable
- ✅ Reconstruction d'état à n'importe quelle date
- ✅ Debug facilité (replay pour reproduire un bug)
- ✅ Conformité réglementaire

### Négatives (trade-offs acceptés)
- ❌ Complexité accrue (CQRS nécessaire pour les requêtes)
- ❌ Courbe d'apprentissage pour l'équipe
- ❌ Migrations d'événements délicates (event versioning)

## Alternatives Considérées

1. **Audit table classique**
   - Rejeté : Reconstruction d'état impossible, données partielles

2. **Soft delete + historique**
   - Rejeté : Ne capture pas toutes les modifications

3. **Temporal database**
   - Rejeté : Vendor lock-in, moins de contrôle

---
Date: 2024-02-15
Auteur: Équipe Core Banking
Révisé: Équipe Architecture
```

---

## Bonnes Pratiques

### Quand Créer un ADR ?

```
✅ CRÉER un ADR pour :

□ Choix de technologie structurant
  → Ex: "Nous utilisons PostgreSQL" (pas "Nous utilisons Python 3.11")

□ Pattern d'architecture adopté
  → Ex: "CQRS pour le reporting", "Saga pour les transactions"

□ Décision impactant plusieurs équipes
  → Ex: "Format standard des événements"

□ Trade-off significatif accepté
  → Ex: "Consistance éventuelle pour les notifications"

□ Décision difficile à reverser
  → Ex: "Migration vers microservices"
```

```
❌ NE PAS créer d'ADR pour :

□ Détails d'implémentation
  → Ex: "Nom des variables", "Ordre des imports"

□ Choix triviaux ou standards
  → Ex: "Utiliser Git pour le versioning"

□ Décisions temporaires
  → Ex: "Workaround pour un bug"
```

### Numérotation et Organisation

```
docs/
└── architecture/
    └── decisions/
        ├── 0001-api-gateway-centralisee.md
        ├── 0002-event-sourcing-policies.md
        ├── 0003-cqrs-reporting.md
        ├── 0007-saga-souscription.md
        └── README.md  (index des ADR)
```

### Cycle de Vie d'un ADR

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   PROPOSÉ    │────▶│   ACCEPTÉ    │────▶│   DÉPRÉCIÉ   │
└──────────────┘     └──────────────┘     └──────────────┘
                                                 │
                                                 ▼
                                          ┌──────────────┐
                                          │  REMPLACÉ    │
                                          │ par ADR-XXX  │
                                          └──────────────┘

⚠️ Un ADR accepté ne se modifie JAMAIS
   → On crée un nouvel ADR qui le supersede
```

---

## ADR Léger (Format Court)

Pour les décisions moins critiques, un format allégé :

```markdown
# ADR-012: Pub/Sub pour les Notifications

**Statut:** Accepté | **Date:** 2024-03-01

**Contexte:** Plusieurs services doivent réagir à PolicyCreated.

**Décision:** Pub/Sub avec topics par type d'événement.

**Conséquences:**
- ✅ Découplage, scalabilité
- ❌ Consistance éventuelle

**Rejeté:** Appels REST directs (couplage), Webhooks (fiabilité).
```

---

## Outils et Automatisation

### Commande pour Créer un ADR

```bash
# Script simple pour créer un nouvel ADR
#!/bin/bash
NEXT_NUM=$(ls docs/adr/*.md | wc -l)
NEXT_NUM=$((NEXT_NUM + 1))
FILENAME=$(printf "docs/adr/%04d-%s.md" $NEXT_NUM "$1")

cat > "$FILENAME" << 'EOF'
# ADR-XXXX: TITRE

## Statut
Proposé

## Contexte
...

## Décision
...

## Conséquences
...

---
Date: $(date +%Y-%m-%d)
Auteur:
EOF

echo "Créé: $FILENAME"
```

---

## Sandbox : Pratiquer

Dans le scénario **CROSS-04**, vous documenterez vos décisions d'architecture en créant des ADR pour :
- Le choix entre orchestration et chorégraphie
- La stratégie de gestion des erreurs
- Le pattern de données choisi

Cette pratique vous familiarisera avec la documentation des décisions.
