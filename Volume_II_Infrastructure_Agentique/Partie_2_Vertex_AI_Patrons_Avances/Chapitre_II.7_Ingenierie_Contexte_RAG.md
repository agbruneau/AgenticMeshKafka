# Chapitre II.7 — Ingénierie du Contexte et RAG

## Introduction

L'un des défis fondamentaux des agents cognitifs réside dans leur capacité à accéder à des informations actualisées, contextuelles et fiables. Les grands modèles de langage (LLM), malgré leurs capacités impressionnantes de génération textuelle, souffrent de limitations intrinsèques : leur connaissance est figée à la date de leur entraînement, ils ne peuvent accéder aux données propriétaires de l'entreprise, et ils sont sujets aux « hallucinations » — ces réponses plausibles mais factuellement incorrectes.

Ces limitations ne sont pas de simples inconvénients techniques. Dans un contexte d'entreprise, une réponse incorrecte d'un assistant IA peut avoir des conséquences significatives : un conseil juridique erroné, une information client obsolète, une recommandation produit inadaptée. La confiance des utilisateurs, condition sine qua non de l'adoption, s'érode rapidement face à des inexactitudes répétées.

La génération augmentée par récupération (RAG — *Retrieval-Augmented Generation*) constitue la réponse architecturale à ces limitations. Ce patron de conception permet d'ancrer les agents dans la réalité en leur fournissant un contexte factuel issu de sources de données vérifiées avant chaque génération de réponse. Plus qu'une simple technique, le RAG représente un changement de paradigme : le passage d'une intelligence paramétrique (encodée dans les poids du modèle) à une intelligence contextuelle (enrichie dynamiquement par des sources externes).

Ce chapitre explore les fondements du RAG, les stratégies avancées d'ingestion et de récupération, et l'intégration critique avec le backbone événementiel Kafka pour maintenir une base de connaissances en temps réel. Nous examinerons également les évolutions récentes vers le RAG agentique, où les systèmes orchestrent dynamiquement leurs stratégies de récupération plutôt que de suivre un flux linéaire prédéfini.

---

## II.7.1 Le Patron RAG : Ancrer les Agents dans la Réalité

### Anatomie du Patron RAG

Le patron RAG repose sur un principe fondamental : plutôt que de s'appuyer uniquement sur la connaissance paramétrique encodée dans les poids du modèle, le système récupère dynamiquement des informations pertinentes à partir d'une base de connaissances externe avant de générer une réponse. Cette approche hybride combine la puissance générative des LLM avec la précision factuelle d'un système de récupération d'information.

L'analogie la plus parlante est celle de l'expert humain consultant sa documentation. Un médecin, même expérimenté, vérifie les interactions médicamenteuses avant de prescrire. Un avocat consulte la jurisprudence récente avant de formuler un avis. Le RAG reproduit ce comportement prudent au niveau algorithmique.

Le flux RAG se décompose en trois phases distinctes :

**Phase 1 — Indexation (préparation)**
Les documents sources sont ingérés, transformés en fragments (*chunks*), puis convertis en représentations vectorielles (embeddings) stockées dans une base de données vectorielle. Cette phase constitue la construction de la « mémoire externe » du système. L'indexation n'est pas un événement unique mais un processus continu dans les architectures modernes, comme nous le verrons dans la section II.7.3.

**Phase 2 — Récupération (au moment de la requête)**
Lorsqu'un utilisateur soumet une question, celle-ci est également convertie en vecteur. Une recherche de similarité identifie les fragments les plus pertinents dans la base vectorielle. Ces fragments constituent le contexte factuel qui sera injecté dans le prompt. La qualité de cette récupération détermine largement la qualité de la réponse finale — un système ne peut générer une bonne réponse s'il récupère des informations non pertinentes.

**Phase 3 — Génération (augmentée)**
Le LLM reçoit un prompt augmenté combinant la question originale et les fragments récupérés. Le prompt suit généralement une structure du type : « Basé sur les informations suivantes [contexte], réponds à la question [question] ». Le modèle génère alors une réponse ancrée dans ce contexte factuel, réduisant significativement les risques d'hallucination.

> **Perspective stratégique**
> Selon Forrester (2025), le RAG est devenu l'architecture par défaut pour les assistants de connaissances d'entreprise. Cette adoption massive reflète un équilibre optimal entre précision factuelle et flexibilité générative — sans nécessiter le coût et la complexité du fine-tuning pour chaque domaine métier. McKinsey rapporte que 71 % des organisations utilisent désormais l'IA générative dans au moins une fonction, et le RAG représente le mécanisme de grounding dominant pour ces déploiements.

### Avantages du RAG par rapport au Fine-Tuning

Le fine-tuning, qui consiste à réentraîner un modèle sur des données spécifiques, présente des inconvénients significatifs dans un contexte d'entreprise. Cette approche était autrefois considérée comme la voie naturelle pour spécialiser un LLM. L'expérience opérationnelle a révélé ses limites.

| Critère | Fine-Tuning | RAG |
|---------|-------------|-----|
| Coût initial | Élevé (GPU, données curées) | Modéré (infrastructure vectorielle) |
| Mise à jour des données | Réentraînement complet | Incrémentale en temps réel |
| Traçabilité des sources | Opaque | Citations et références possibles |
| Délai de déploiement | Semaines à mois | Heures à jours |
| Risque de « catastrophic forgetting » | Élevé | Inexistant |
| Multi-domaines | Un modèle par domaine | Une infrastructure pour tous |

Le fine-tuning souffre particulièrement du phénomène de « catastrophic forgetting » : l'entraînement sur de nouvelles données peut dégrader les capacités du modèle sur les connaissances antérieures. Cette instabilité rend difficile la maintenance d'un modèle fine-tuné sur des données évolutives.

Le RAG excelle particulièrement dans les scénarios où les données évoluent fréquemment (prix, inventaires, politiques), où la traçabilité des sources est critique (conformité réglementaire, audit), et où plusieurs domaines métier doivent être servis par une infrastructure commune (service client multi-produits).

### Limites du RAG Naïf

Le RAG « naïf » — une implémentation basique avec découpage fixe et recherche vectorielle simple — présente des limitations documentées qui ont motivé le développement des approches avancées :

**Perte de contexte** : Le découpage arbitraire fragmente la narration et peut séparer des informations interdépendantes. Une phrase peut mentionner « il » sans que le référent soit inclus dans le chunk récupéré. Cette fragmentation dégrade la compréhension globale.

**Problème « Lost in the Middle »** : Des recherches (Liu et al., 2023) ont démontré que les LLM accordent plus d'attention aux premiers et derniers éléments du contexte, négligeant les informations centrales. Un document pertinent positionné au milieu d'un contexte long peut être ignoré lors de la génération.

**Échec sur les requêtes complexes** : Les questions nécessitant un raisonnement multi-étapes (« Quel est l'impact financier combiné des projets A et B sur la région X ? ») ou la synthèse de sources multiples restent difficiles pour le RAG naïf, qui récupère des fragments sans considérer leur interconnexion.

**Manque de connaissance globale** : Le RAG récupère des fragments isolés sans vision d'ensemble du corpus. Les questions de type « Quels sont les principaux thèmes abordés dans notre documentation ? » ne peuvent être traitées correctement car elles nécessitent une compréhension holistique que la récupération fragmentaire ne permet pas.

**Inadéquation terminologique** : Une recherche vectorielle pour « rentabilité » peut manquer des documents parlant de « ROI » ou « retour sur investissement » si les embeddings ne capturent pas parfaitement ces synonymies. Ce problème est particulièrement aigu pour les terminologies techniques spécifiques à un domaine.

Ces limitations motivent l'adoption de stratégies RAG avancées que nous explorerons dans la section II.7.4.

---

## II.7.2 Gestion de la Mémoire Vectorielle

### Fondements des Embeddings

Les embeddings constituent le cœur du système RAG. Ces représentations vectorielles denses capturent le sens sémantique du texte dans un espace mathématique à haute dimension (typiquement 768 à 4096 dimensions). Deux textes sémantiquement proches se retrouvent géométriquement proches dans cet espace, permettant une recherche par similarité efficace.

Le processus de création d'embeddings repose sur des modèles pré-entraînés sur de vastes corpus textuels. Ces modèles apprennent à projeter le texte dans un espace où les relations sémantiques sont préservées : « roi - homme + femme ≈ reine » illustre classiquement cette propriété.

La qualité des embeddings influence directement la pertinence des récupérations. Un modèle d'embedding médiocre positionne des textes non reliés à proximité, ou éloigne des textes reliés, dégradant la qualité du RAG indépendamment de la sophistication des autres composants.

Plusieurs familles de modèles d'embedding coexistent en 2025, chacune avec ses forces et compromis :

**Modèles propriétaires (API)**
- OpenAI text-embedding-3-large/small : Compréhension sémantique améliorée avec dimensions configurables (256 à 3072). La version « small » offre un excellent rapport qualité/coût pour les déploiements à haut volume.
- Cohere Embed v4 : Optimisé pour les langues multiples avec support de plus de 100 langues et options de compression pour réduire les coûts de stockage.
- Google Vertex AI text-embedding-005 : Intégration native avec l'écosystème GCP, supportant jusqu'à 2048 tokens d'entrée et produisant des vecteurs de 768 dimensions.

**Modèles open source**
- E5-small/base/large-instruct : Performance exceptionnelle avec faible latence (< 30 ms). Les benchmarks démontrent 100 % de précision Top-5 pour E5-small, surpassant des modèles beaucoup plus grands.
- BGE (BAAI General Embedding) : Leader sur les benchmarks MTEB, particulièrement performant sur les tâches de récupération asymétrique (questions courtes vers documents longs).
- Mistral Embed : Précision Top-1 de 77,8 % dans les benchmarks, le positionnant comme choix optimal quand la précision de rang est critique.
- Llama-embed-nemotron-8b : Meilleure précision Top-1 (62 %) parmi les modèles testés sur les benchmarks RAG spécifiques, mais avec une latence plus élevée (~200 ms).

> **Bonnes pratiques**
> Pour les systèmes RAG de production, les modèles E5-small et E5-base-instruct combinent haute précision (100 % Top-5 sur les benchmarks) et faible latence (moins de 30 ms). Privilégiez ces modèles sauf si des contraintes spécifiques (multilinguisme, conformité, souveraineté) imposent d'autres choix. Le fine-tuning des embeddings sur des données domaine-spécifiques améliore typiquement la précision de 20 à 40 % pour les vocabulaires techniques.

### Choix du Modèle d'Embedding dans Vertex AI

Vertex AI RAG Engine supporte plusieurs options d'embedding, permettant d'adapter le choix aux contraintes spécifiques du projet. Le tableau suivant résume les choix disponibles :

| Modèle | Dimensions | Tokens max | Cas d'usage |
|--------|------------|------------|-------------|
| text-embedding-005 (Gecko) | 768 | 2048 | Usage général, intégration native GCP |
| text-multilingual-embedding-002 | 768 | 2048 | Corpus multilingues, contenu international |
| Modèles fine-tunés | 768 | Variable | Domaines spécialisés (médical, juridique, financier) |
| Modèles open source déployés | Variable | Variable | Souveraineté des données, coûts optimisés |

L'option de fine-tuning mérite une attention particulière. Vertex AI permet de spécialiser le modèle Gecko sur un corpus propriétaire, améliorant la capture des terminologies métier spécifiques. Cette spécialisation est particulièrement bénéfique pour les domaines à vocabulaire technique dense (médical, juridique, ingénierie).

### Bases de Données Vectorielles

Le choix de la base de données vectorielle impacte directement les performances du système RAG. Les critères d'évaluation incluent la latence de requête (typiquement < 100 ms pour les applications interactives), la scalabilité (millions à milliards de vecteurs), les capacités de filtrage hybride (combinaison de filtres metadata et recherche vectorielle), et l'intégration écosystémique.

**Solutions managées**

*Vertex AI RagManagedDb* constitue l'option par défaut dans l'écosystème Google Cloud. Basée sur Spanner, elle élimine la gestion d'infrastructure tout en offrant une scalabilité automatique. La facturation active depuis novembre 2025 impose une planification budgétaire, mais le modèle « pay-as-you-go » reste attractif pour les déploiements progressifs.

*Pinecone* représente le leader du marché en serverless vector database. Son architecture découple calcul et stockage, permettant un scaling indépendant. L'entreprise rapporte des temps de requête inférieurs à 50 ms même sur des index de milliards de vecteurs.

*Weaviate Cloud* se distingue par son API GraphQL native et ses modules de vectorisation intégrés. Cette approche permet d'encapsuler l'appel au modèle d'embedding dans la base de données, simplifiant l'architecture globale.

**Solutions auto-hébergées**

*Qdrant*, développé en Rust, excelle en performance temps réel avec un filtrage JSON riche. Son architecture permet des requêtes combinant similarité vectorielle et filtres complexes sur les métadonnées sans dégradation de performance.

*Milvus* (commercialisé en SaaS sous le nom Zilliz Cloud) offre une accélération GPU native, le positionnant pour les très grands volumes (centaines de millions de vecteurs) et les exigences de débit élevé.

*pgvector* étend PostgreSQL avec des capacités vectorielles. Cette approche séduit les organisations souhaitant consolider leurs données vectorielles avec leur infrastructure relationnelle existante, réduisant la complexité opérationnelle.

**Solutions hybrides GCP**

*AlloyDB for PostgreSQL* combine la compatibilité PostgreSQL avec une performance vectorielle optimisée par Google. Cette option convient aux organisations ayant une expertise PostgreSQL établie.

*Cloud SQL with pgvector* constitue l'option économique pour les volumes modérés, bien que les performances se dégradent au-delà de quelques millions de vecteurs.

> **Note technique**
> Vertex AI RAG Engine utilise Spanner comme base vectorielle managée (GA avec facturation active). Pour les nouveaux projets sans contraintes spécifiques, cette option élimine la complexité opérationnelle tout en offrant une scalabilité transparente. Pour les projets nécessitant un contrôle fin ou des performances extrêmes, Qdrant ou Pinecone restent des alternatives de premier plan.

### Recherche Hybride et Reranking

La recherche vectorielle pure présente des faiblesses sur les requêtes contenant des termes techniques, des acronymes, des identifiants produit ou des codes spécifiques. La phrase « Problème avec le SKU ABC-123 » nécessite une correspondance exacte que la similarité sémantique peut manquer.

La recherche hybride combine deux approches complémentaires :

**Recherche dense (vectorielle)** : Capture le sens sémantique. « Comment retourner un produit ? » trouvera « Procédure de remboursement » même si les mots diffèrent.

**Recherche sparse (lexicale, BM25)** : Identifie les correspondances exactes de mots-clés. Le terme « ABC-123 » sera retrouvé par correspondance exacte.

La fusion des résultats s'effectue typiquement via Reciprocal Rank Fusion (RRF), qui pondère les scores de chaque méthode selon la formule : score_final = Σ 1/(k + rang_i) où k est un paramètre de lissage (typiquement 60).

Le **reranking** constitue une seconde étape critique. Après une récupération large (top-25 à top-50), un modèle de cross-encoder évalue la pertinence fine de chaque paire (requête, document). Contrairement aux bi-encoders (qui encodent requête et document séparément), les cross-encoders analysent la paire conjointement, capturant des interactions subtiles.

Vertex AI propose un service de reranking intégré qui améliore significativement le classement final. Le flux optimisé devient :

```
Requête → Recherche hybride (top-50) → Reranker (top-5) → Contexte LLM
```

Cette approche en deux étapes équilibre rappel (retrieval large) et précision (reranking fin) de manière optimale.

---

## II.7.3 Ingestion des Données en Temps Réel pour le RAG via Kafka

### L'Impératif du Temps Réel

La valeur d'un système RAG dépend directement de la fraîcheur de sa base de connaissances. Un RAG alimenté par des données obsolètes produit des réponses incorrectes malgré une architecture sophistiquée. Cette réalité impose une intégration étroite entre le backbone événementiel et la couche cognitive.

Considérons un scénario concret : un client contacte le support pour une commande. Si la base de connaissances RAG ne reflète pas l'annulation effectuée il y a 10 minutes, l'agent IA fournira des informations incorrectes, érodant la confiance et nécessitant une escalade humaine coûteuse.

Apache Kafka, en tant que colonne vertébrale du système nerveux numérique (cf. chapitres II.2-II.5), fournit l'infrastructure idéale pour maintenir une base de connaissances RAG en synchronisation continue avec les données opérationnelles. Son modèle de journal immuable et ses garanties de livraison en font le choix naturel pour ce type d'intégration.

> **Perspective stratégique**
> L'intégration Kafka-RAG transforme la base de connaissances d'un snapshot statique en un « miroir vivant » de l'entreprise. Cette architecture permet aux agents cognitifs de répondre sur la base de l'état actuel des systèmes, et non sur une photo figée potentiellement périmée. Expedia, par exemple, utilise cette architecture pour ses chatbots de service client, garantissant que les informations de vol reflètent l'état en temps réel.

### Architecture d'Ingestion Streaming

L'architecture d'ingestion RAG temps réel s'articule autour de trois composants majeurs, chacun jouant un rôle spécifique dans la chaîne de valeur :

**1. Capture des changements (CDC)**

Debezium ou Kafka Connect capturent les modifications des bases de données sources. Chaque INSERT, UPDATE ou DELETE génère un événement sur un topic Kafka dédié. Cette approche « event sourcing » garantit que toute modification est immédiatement disponible pour traitement.

La configuration Debezium typique pour une base PostgreSQL :

```yaml
name: "postgres-cdc-source"
connector.class: "io.debezium.connector.postgresql.PostgresConnector"
database.hostname: "db.example.com"
database.port: "5432"
database.user: "debezium"
database.dbname: "knowledge_base"
table.include.list: "public.documents,public.articles,public.policies"
topic.prefix: "rag-ingest"
publication.name: "dbz_publication"
slot.name: "debezium_slot"
```

Cette configuration capture les changements des tables pertinentes et les publie sur des topics Kafka avec le préfixe `rag-ingest`.

**2. Transformation et chunking (Flink/Kafka Streams)**

Les événements bruts nécessitent une préparation avant indexation. Apache Flink ou Kafka Streams effectuent plusieurs opérations critiques :

*Nettoyage et normalisation* : Suppression des balises HTML, normalisation des encodages, extraction du texte des formats structurés.

*Découpage sémantique* : Application de la stratégie de chunking appropriée au type de document (récursif pour les articles, hiérarchique pour les manuels).

*Enrichissement avec métadonnées* : Ajout du timestamp de modification, de la source système, de l'auteur, des tags de classification.

Flink SQL permet de définir ces transformations de manière déclarative :

```sql
CREATE TABLE document_chunks AS
SELECT 
    doc_id,
    chunk_id,
    chunk_text,
    CURRENT_TIMESTAMP as indexed_at,
    source_system,
    author,
    document_type
FROM TABLE(
    SEMANTIC_CHUNK(
        document_text,
        512,   -- taille cible en tokens
        50,    -- overlap en tokens
        document_type  -- adapte la stratégie
    )
);
```

**3. Génération d'embeddings et indexation**

Les chunks préparés sont envoyés à un service d'embedding (Vertex AI, OpenAI, ou modèle auto-hébergé). Confluent Cloud offre l'AI Model Inference natif, permettant d'appeler les modèles directement depuis Flink SQL :

```sql
CREATE TABLE chunk_embeddings AS
SELECT 
    chunk_id,
    chunk_text,
    ML_PREDICT(
        'projects/my-project/locations/us-central1/endpoints/embedding-endpoint',
        chunk_text
    ) as embedding_vector,
    indexed_at,
    source_system
FROM document_chunks;
```

Les vecteurs résultants sont indexés dans la base vectorielle via un sink connector (Qdrant, Pinecone, ou Vertex AI Vector Search).

```
[Sources] → [Debezium/CDC] → [Kafka Topics] → [Flink Processing]
                                                      ↓
                                              [Embedding Service]
                                                      ↓
                                              [Vector DB Sink]
```

### Gestion de la Cohérence et du Cycle de Vie

L'ingestion streaming introduit des considérations spécifiques de cohérence et de gestion du cycle de vie des documents :

**Gestion des mises à jour**

Un document modifié doit déclencher la suppression des anciens chunks et l'indexation des nouveaux. L'utilisation d'identifiants stables (document_id + version) permet de gérer ce cycle de vie. La séquence typique est :
1. Réception de l'événement UPDATE
2. Suppression des chunks existants avec le document_id
3. Chunking du nouveau contenu
4. Indexation des nouveaux chunks

**Fenêtres de traitement**

Pour les documents volumineux ou les rafales d'événements (import batch, migration), le traitement par fenêtres (*windowing*) évite de surcharger le service d'embedding. Une fenêtre de 100 événements ou 10 secondes offre un bon compromis latence/efficacité. Flink gère nativement ces fenêtres avec ses opérateurs temporels.

**Monitoring de la fraîcheur**

Un indicateur critique est le *consumer lag* — le décalage entre les événements produits et consommés. Un lag croissant signale un goulot d'étranglement dans la pipeline d'embedding. Ce lag doit être monitoré via les métriques JMX de Kafka et alerté au-delà d'un seuil acceptable (typiquement quelques secondes pour les applications temps réel).

> **Attention**
> La génération d'embeddings via API introduit une latence de 100 à 500 ms par document. À haut volume (> 10 000 documents/heure), cette latence nécessite une parallélisation agressive (multiple partitions Kafka, plusieurs instances consumer) et potentiellement une mise en cache des embeddings pour les documents inchangés. Le coût API peut également devenir significatif à grande échelle, justifiant l'évaluation de modèles auto-hébergés.

---

## II.7.4 Stratégies Avancées de RAG

### Stratégies de Chunking

Le découpage des documents constitue l'un des leviers les plus impactants sur la qualité du RAG. La recherche de 2025 établit un consensus empirique : la taille optimale se situe entre 256 et 512 tokens, avec un overlap de 10 à 20 % pour préserver le contexte aux frontières.

**Chunking à taille fixe**

Découpage naïf basé sur un nombre de caractères ou tokens. Simple à implémenter mais ignore la structure sémantique. Un chunk peut commencer au milieu d'une phrase et terminer au milieu d'une autre, perdant toute cohérence. Cette approche reste acceptable uniquement pour les contenus très homogènes (logs, données tabulaires converties en texte).

**Chunking récursif**

Respecte les frontières naturelles (paragraphes, sections, phrases) tout en maintenant une taille cible. L'algorithme tente d'abord de diviser par paragraphes, puis par phrases si les paragraphes sont trop longs, puis par mots en dernier recours. LangChain popularise cette approche avec son RecursiveCharacterTextSplitter.

**Chunking sémantique**

Utilise des modèles pour identifier les ruptures thématiques au sein du texte. Produit des chunks cohérents sémantiquement mais de taille variable. Cette variabilité peut poser des défis pour le stockage vectoriel (certaines bases optimisent pour des tailles uniformes) mais améliore significativement la qualité de récupération.

**Chunking hiérarchique (Parent-Child)**

Indexe des chunks petits pour la précision de récupération, mais retourne le chunk parent (plus large) pour préserver le contexte lors de la génération. Par exemple, on indexe des paragraphes individuels, mais quand un paragraphe est récupéré, on retourne la section entière. Cette stratégie atténue élégamment le problème de fragmentation.

**Chunking adaptatif**

Ajuste la stratégie selon le type de document. Un contrat juridique (sections numérotées, références croisées) ne se découpe pas comme une FAQ (questions-réponses indépendantes) ou un manuel technique (procédures étape par étape). L'identification automatique du type de document permet d'appliquer la stratégie optimale.

> **Bonnes pratiques**
> Pour les documents techniques multi-thématiques (manuels, rapports), le chunking hiérarchique offre le meilleur compromis. Pour les documents courts et focalisés (FAQ, tickets support), le chunking document-level ou minimal est souvent préférable — parfois, le document entier constitue le meilleur chunk.

### Transformation de Requêtes

La formulation d'une question par l'utilisateur ne correspond pas toujours à la manière dont l'information est structurée dans les documents. Un utilisateur demande « Comment annuler ma commande ? » alors que la documentation parle de « Procédure de rétractation ». Les techniques de transformation de requêtes comblent cet écart.

**HyDE (Hypothetical Document Embeddings)**

Génère une réponse hypothétique à la question, puis utilise son embedding pour la recherche. L'intuition : l'embedding d'une réponse hypothétique est plus proche des documents pertinents que l'embedding de la question seule. Cette technique améliore significativement le rappel pour les requêtes vagues ou mal formulées.

**Query Expansion**

Enrichit la requête avec des synonymes, termes connexes ou variantes. La question « rentabilité du projet » devient « rentabilité ROI retour investissement marge projet ». Augmente le rappel sans sacrifier excessivement la précision quand combiné avec reranking.

**Step-back Prompting**

Pour les questions très spécifiques, génère d'abord une question plus générale. « Quelle est la limite de retour pour les écouteurs sans fil ? » génère d'abord « Quelle est la politique de retour ? », récupère ce contexte large, puis répond à la question spécifique. Cette approche imite le raisonnement humain qui contextualise avant de détailler.

**Décomposition de requêtes**

Les questions complexes multi-parties sont décomposées en sous-questions, chacune traitée indépendamment puis synthétisée. « Comparez les performances de nos produits A et B sur les marchés européen et asiatique » devient quatre requêtes séparées, dont les résultats sont agrégés pour la réponse finale.

### Graph RAG

Le GraphRAG adresse une limitation fondamentale du RAG traditionnel : l'incapacité à capturer les relations entre entités et à effectuer un raisonnement multi-étapes. Les approches vectorielles traitent les documents comme des unités isolées, ignorant leurs interconnexions.

**Principe architectural**

Un graphe de connaissances est construit à partir du corpus, représentant les entités (personnes, produits, concepts, événements) et leurs relations (travaille pour, contient, précède). La récupération ne se limite plus aux passages textuels mais inclut les chemins relationnels dans le graphe.

L'implémentation de Microsoft GraphRAG procède en plusieurs étapes :
1. Extraction d'entités et relations via LLM
2. Construction du graphe et détection de communautés
3. Génération de résumés par communauté
4. Indexation des entités, relations et résumés

**Modes de requête**

*Local Search* : Pour les questions sur des entités spécifiques, le système navigue le graphe à partir de l'entité identifiée, collectant les relations pertinentes.

*Global Search* : Pour les questions holistiques (« Quels sont les principaux thèmes ? »), le système exploite les résumés de communautés plutôt que les documents individuels.

**Avantages documentés**

- Capture les relations structurées que la similarité sémantique ignore
- Permet le raisonnement multi-hop (« Qui dirige le département responsable du projet X ? »)
- Offre une meilleure explicabilité via la traçabilité des chemins relationnels

> **Note technique**
> Google Cloud propose GraphRAG via Spanner Graph. Cette option combine la scalabilité de Spanner avec les capacités de requête graph pour les applications RAG d'entreprise. L'intégration native avec Vertex AI simplifie le déploiement.

### RAG Agentique

L'évolution la plus significative du RAG en 2025 est son intégration dans des workflows agentiques. Plutôt qu'un flux linéaire prédéfini (récupération → génération), les systèmes RAG agentiques orchestrent dynamiquement leurs stratégies de récupération selon la nature de chaque requête.

**Self-RAG**

Le modèle décide lui-même quand récupérer des informations supplémentaires, évalue la pertinence des documents récupérés, et critique ses propres réponses. Des « tokens de réflexion » spéciaux (retrieve, relevant, supported) guident ce processus d'auto-correction intégré au modèle.

**Corrective RAG (CRAG)**

Introduit un mécanisme de rétroaction corrective explicite. Un évaluateur (qui peut être un autre LLM ou un classifieur dédié) note la pertinence des documents récupérés sur une échelle (correct, ambigu, incorrect). Si le score est insuffisant, le système déclenche automatiquement une recherche alternative — web search, autre base de connaissances, ou reformulation de requête.

**Adaptive RAG**

Classifie la complexité de chaque requête avant de déterminer la stratégie. Une question factuelle simple (« Quels sont nos horaires d'ouverture ? ») peut être traitée directement depuis le cache ou la base de connaissances sans processus RAG complet. Une question complexe déclenche un workflow multi-étapes avec décomposition et synthèse.

**Architecture Multi-Agents pour RAG**

Des agents spécialisés collaborent dans le workflow RAG, chacun optimisé pour sa tâche :
- *Agent de routage* : Analyse la requête et détermine la stratégie optimale
- *Agent de récupération* : Exécute les recherches (vectorielle, hybride, graph)
- *Agent d'évaluation* : Valide la pertinence et la complétude du contexte
- *Agent de synthèse* : Génère la réponse finale avec citations

Cette architecture permet une spécialisation des modèles (petits modèles rapides pour le routage et l'évaluation, grands modèles pour la synthèse) optimisant le rapport qualité/coût.

### Évaluation et Métriques RAG

L'amélioration d'un système RAG nécessite une instrumentation rigoureuse. Sans métriques, les optimisations restent aveugles. Les métriques clés se répartissent en deux catégories :

**Métriques de récupération**

| Métrique | Description | Cible typique |
|----------|-------------|---------------|
| Hit Rate / Recall@K | Le document pertinent apparaît-il dans le top-K ? | > 90 % |
| MRR (Mean Reciprocal Rank) | Rang moyen du premier document pertinent | > 0.7 |
| Precision@K | Proportion de documents pertinents dans le top-K | > 60 % |

**Métriques de génération**

| Métrique | Description | Cible typique |
|----------|-------------|---------------|
| Faithfulness / Groundedness | Affirmations supportées par le contexte | > 90 % |
| Citation Precision | Citations pointant vers les faits énoncés | > 85 % |
| Answer Relevance | Réponse adressant la question posée | > 90 % |

Des frameworks comme RAGAS, DeepEval ou LangSmith automatisent ces évaluations via des juges LLM. L'intégration dans les pipelines CI/CD permet une régression continue de la qualité à chaque modification du système.

> **Attention**
> Une amélioration du rappel sans reranking peut paradoxalement dégrader la qualité finale. Le LLM reçoit alors plus de contexte, mais potentiellement plus de bruit qui dilue les informations pertinentes. Toujours évaluer l'impact end-to-end (métriques de génération), pas seulement les métriques de récupération isolées.

---

## II.7.5 Résumé

Ce chapitre a exploré l'ingénierie du contexte et les architectures RAG, fondements essentiels pour ancrer les agents cognitifs dans la réalité factuelle de l'entreprise.

### Points clés

**Le patron RAG**
- Combine récupération d'information et génération pour réduire drastiquement les hallucinations
- Trois phases : indexation (préparation de la mémoire), récupération (au moment de la requête), génération augmentée
- Avantages décisifs sur le fine-tuning : mise à jour incrémentale, traçabilité des sources, déploiement rapide, pas de catastrophic forgetting

**Mémoire vectorielle**
- Les embeddings capturent le sens sémantique dans un espace vectoriel à haute dimension
- Choix critiques : modèle d'embedding (E5, Gecko, BGE pour la performance ; fine-tuning pour les domaines spécialisés) et base vectorielle (RagManagedDb pour la simplicité, Qdrant/Pinecone pour le contrôle)
- La recherche hybride (dense + sparse) et le reranking améliorent significativement la précision au-delà de la recherche vectorielle pure

**Ingestion temps réel via Kafka**
- CDC (Debezium) capture les changements à la source avec garantie de livraison
- Flink/Kafka Streams transforment, préparent et enrichissent les chunks
- Pipeline d'embedding et indexation vectorielle en continu
- Monitoring du consumer lag critique pour garantir la fraîcheur des données

**Stratégies RAG avancées**
- Chunking : hiérarchique et adaptatif pour préserver le contexte selon le type de document
- Transformation de requêtes : HyDE, expansion, step-back, décomposition pour combler l'écart terminologique
- Graph RAG : capture les relations structurées et permet le raisonnement multi-hop impossible en RAG vectoriel pur
- RAG agentique : Self-RAG, CRAG, Adaptive RAG pour l'auto-correction et l'orchestration dynamique

### Recommandations architecturales

| Composant | Recommandation |
|-----------|----------------|
| Embedding | E5-base-instruct (performance/latence) ou Vertex AI text-embedding-005 (intégration GCP) |
| Base vectorielle | RagManagedDb (simplicité opérationnelle) ou Qdrant (contrôle et performance) |
| Chunking | Hiérarchique parent-child (256-512 tokens, 10-20% overlap) |
| Recherche | Hybride (dense + BM25 via RRF) avec reranking Vertex AI ou cross-encoder |
| Ingestion | Pipeline Kafka → Flink → Embedding Service → Vector DB avec monitoring lag |
| Évaluation | RAGAS ou DeepEval intégré au CI/CD avec métriques retrieval et generation |

### Transition vers le chapitre suivant

L'ingénierie du contexte via RAG constitue le premier pilier de l'ancrage des agents dans la réalité. Le chapitre suivant (II.8) explorera l'intégration complète du backbone événementiel Confluent avec la couche cognitive Vertex AI, démontrant comment ces composants s'assemblent pour créer une architecture agentique cohérente et opérationnelle. Nous examinerons en particulier l'orchestration des agents sur les flux d'événements et la réalisation pratique du concept de Jumeau Numérique Cognitif (JNC) introduit au Volume I.

---

*Ce chapitre s'appuie sur les dernières avancées documentées en matière de RAG, incluant les travaux sur Self-RAG (Asai et al., 2023), CRAG (Yan et al., 2024), GraphRAG (Microsoft Research, 2024), et les architectures de production décrites par Confluent, Google Cloud et les leaders de l'industrie. Les benchmarks cités proviennent d'évaluations indépendantes réalisées en 2025.*

*Chapitre suivant : Chapitre II.8 — Intégration du Backbone Événementiel et de la Couche Cognitive*
