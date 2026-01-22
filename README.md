# Interopérabilité en Écosystème d'Entreprise

**Convergence des Architectures d'Intégration — Du Couplage Fort au Découplage Maximal**

Une monographie technique explorant le continuum d'intégration d'entreprise à travers trois domaines : Applications, Données et Événements.

## Aperçu

Ce projet est un site web éducatif présentant une monographie complète sur l'interopérabilité en contexte d'entreprise. Le contenu est organisé en 11 chapitres et des annexes, couvrant les fondements théoriques, les patrons d'architecture et les perspectives futures de l'intégration d'entreprise.

### Thèse Centrale

> « L'interopérabilité n'est pas un état binaire qu'on atteint ou qu'on manque, mais un **continuum** qui s'étend du couplage fort au découplage maximal. »

### Les Trois Domaines d'Intégration

| Domaine | Métaphore | Focus | Chapitre |
|---------|-----------|-------|----------|
| **Applications** | Le Verbe | Orchestration, interactions synchrones | III |
| **Données** | Le Nom | Cohérence de l'état, accessibilité | IV |
| **Événements** | Le Signal | Réactivité, découplage temporel maximal | V |

## Structure du Contenu

```
Chapitres/
├── I.   Introduction et Problématique
├── II.  Fondements Théoriques
├── III. Intégration des Applications (Le Verbe)
├── IV.  Intégration des Données (Le Nom)
├── V.   Intégration des Événements (Le Signal)
├── VI.  Standards et Contrats d'Interface
├── VII. Résilience et Observabilité
├── VIII.Collaboration et Automatisation
├── IX.  Architecture de Référence Convergente
├── X.   Étude de Cas : Order-to-Cash
├── XI.  Vers l'Entreprise Agentique
└── Annexes (Glossaire, Comparatifs, Bibliographie)
```

## Technologies

- **[Astro](https://astro.build/)** — Framework de génération de sites statiques
- **[Tailwind CSS](https://tailwindcss.com/)** — Framework CSS utilitaire
- **[@tailwindcss/typography](https://tailwindcss.com/docs/typography-plugin)** — Plugin pour le rendu de prose

## Installation

```bash
# Cloner le dépôt
git clone https://github.com/agbruneau/Experimentation.git
cd Experimentation

# Installer les dépendances
npm install
```

## Commandes

| Commande | Description |
|----------|-------------|
| `npm run dev` | Lancer le serveur de développement sur `localhost:4321` |
| `npm run build` | Construire le site pour la production dans `./dist/` |
| `npm run preview` | Prévisualiser le build localement |

## Développement

```bash
# Lancer le serveur de développement
npm run dev

# Ouvrir dans le navigateur
# http://localhost:4321/Experimentation/
```

Le site supporte le rechargement à chaud (HMR) — les modifications sont reflétées instantanément.

## Déploiement

Le site est automatiquement déployé sur **GitHub Pages** via GitHub Actions à chaque push sur la branche `main`.

**URL de production** : https://agbruneau.github.io/Experimentation/

### Configuration GitHub Pages

1. Aller dans **Settings** → **Pages**
2. Sous **Source**, sélectionner **GitHub Actions**
3. Le workflow `.github/workflows/deploy.yml` gère le déploiement automatique

## Structure du Projet

```
├── src/
│   ├── components/       # Composants Astro réutilisables
│   ├── content/
│   │   └── chapters/     # Fichiers Markdown des chapitres
│   ├── layouts/          # Layouts de page
│   ├── pages/            # Routes du site
│   └── styles/           # Styles globaux
├── public/               # Assets statiques (images, PDF)
├── Chapitres/            # Sources Markdown originales
├── Presentation/         # Poster et présentation PDF
├── astro.config.mjs      # Configuration Astro
├── tailwind.config.mjs   # Configuration Tailwind (thème dark)
└── package.json
```

## Thème Visuel

Le site utilise un thème sombre "Midnight Scholar" optimisé pour la lecture prolongée :

- **Fond** : `#0f0f12` (noir profond)
- **Le Verbe** : `#7c3aed` (violet)
- **Le Nom** : `#06b6d4` (cyan)
- **Le Signal** : `#f59e0b` (ambre)

## Licence

Ce contenu est destiné à des fins éducatives.

## Auteur

Alain G. Bruneau

---

*Généré avec Astro et déployé sur GitHub Pages*
