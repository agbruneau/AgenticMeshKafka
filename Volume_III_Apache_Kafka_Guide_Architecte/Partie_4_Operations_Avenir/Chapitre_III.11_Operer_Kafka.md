# Chapitre III.11 - Opérer Kafka

---

## Introduction

L'exploitation d'un cluster Kafka en production représente un défi permanent qui exige vigilance, expertise et méthodologie. Contrairement à de nombreuses technologies où le déploiement initial constitue l'essentiel de l'effort, Kafka révèle sa complexité dans la durée : évolutions de versions, croissance des volumes, incidents imprévus et exigences de continuité d'activité testent quotidiennement les équipes opérationnelles.

Ce chapitre adopte la perspective de l'architecte responsable de la pérennité opérationnelle d'une plateforme Kafka. Il couvre les cinq dimensions critiques de l'exploitation : l'évolution et les mises à niveau du cluster, la mobilité des données entre environnements et régions, la surveillance approfondie, l'optimisation continue des performances, et les stratégies de reprise après sinistre.

L'enjeu dépasse la simple disponibilité technique. Un cluster Kafka mal opéré accumule une dette opérationnelle qui compromet progressivement sa capacité à servir les besoins métier. Les mises à niveau retardées créent des vulnérabilités de sécurité. Les problèmes de performance non diagnostiqués érodent la confiance des équipes applicatives. Les lacunes dans la préparation aux sinistres se révèlent au pire moment. À l'inverse, une exploitation rigoureuse transforme Kafka en infrastructure de confiance sur laquelle l'organisation peut bâtir sa stratégie événementielle.

---

## III.11.1 Évolution et Mises à Niveau du Cluster

### III.11.1.1 Stratégie de Gestion des Versions

La gestion des versions Kafka constitue un équilibre délicat entre stabilité opérationnelle et accès aux nouvelles fonctionnalités. Apache Kafka maintient une cadence de release soutenue, avec des versions majeures annuelles et des versions mineures trimestrielles.

**Cycle de vie des versions Kafka**

| Type de version | Fréquence    | Contenu                                                          | Support typique               |
| --------------- | ------------- | ---------------------------------------------------------------- | ----------------------------- |
| Majeure (X.0)   | Annuelle      | Nouvelles fonctionnalités majeures, breaking changes potentiels | 2-3 ans                       |
| Mineure (X.Y)   | Trimestrielle | Fonctionnalités incrémentales, améliorations                  | 1 an                          |
| Patch (X.Y.Z)   | Ad hoc        | Corrections de bugs, vulnérabilités                            | Jusqu'à la prochaine mineure |

**Politique de mise à niveau recommandée**

L'architecte doit définir une politique claire qui équilibre risque et bénéfice :

```yaml
# Exemple de politique de mise à niveau

upgrade_policy:
  security_patches:
    delay: 7_days          # Validation minimale en staging
    approval: ops_team
    rollout: immediate_after_validation
  
  minor_versions:
    delay: 30_days         # Attendre retours communauté
    approval: architecture_review
    rollout: quarterly_maintenance_window
  
  major_versions:
    delay: 90_days         # Validation approfondie
    approval: steering_committee
    rollout: planned_project
    prerequisites:
      - full_regression_testing
      - performance_baseline
      - rollback_plan_validated
      - documentation_updated
```

> **Décision architecturale**
>
> *Contexte* : Une institution financière utilise Kafka 3.4 et doit décider de la mise à niveau vers Kafka 3.7 (dernière version stable).
>
> *Options* : (1) Mise à niveau immédiate pour bénéficier des améliorations KRaft, (2) Attente de la version 4.0 pour une migration majeure unique.
>
> *Décision* : Mise à niveau vers 3.7 avec migration KRaft planifiée. La dette technique accumulée en retardant les mises à niveau dépasse le coût d'une migration intermédiaire. De plus, les correctifs de sécurité de 3.4 arrivent en fin de support.

### III.11.1.2 Migration de ZooKeeper vers KRaft

La migration vers KRaft (Kafka Raft) représente l'évolution architecturale majeure de Kafka depuis sa création. KRaft élimine la dépendance à ZooKeeper en intégrant la gestion des métadonnées directement dans les brokers Kafka.

**Avantages de KRaft**

| Dimension                | ZooKeeper                     | KRaft                  | Bénéfice                     |
| ------------------------ | ----------------------------- | ---------------------- | ------------------------------ |
| Architecture             | Cluster séparé à maintenir | Intégré aux brokers  | Simplification opérationnelle |
| Scalabilité             | Limite ~200K partitions       | Millions de partitions | Croissance sans contrainte     |
| Temps de récupération  | Minutes (élection leader)    | Secondes               | Disponibilité améliorée     |
| Complexité déploiement | Double cluster                | Cluster unique         | Réduction des coûts          |

**Processus de migration**

La migration s'effectue en plusieurs phases pour minimiser les risques :

```
Phase 1: Préparation
├── Validation version Kafka ≥ 3.3
├── Audit des configurations ZooKeeper-spécifiques
├── Tests en environnement isolé
└── Formation des équipes

Phase 2: Mode hybride
├── Ajout des contrôleurs KRaft
├── Migration des métadonnées
├── Validation fonctionnelle
└── Surveillance intensive

Phase 3: Basculement
├── Désactivation de ZooKeeper
├── Reconfiguration des clients (si nécessaire)
├── Décommissionnement ZooKeeper
└── Documentation mise à jour
```

**Script de migration KRaft**

```bash
#!/bin/bash
# scripts/migrate_to_kraft.sh

KAFKA_HOME="/opt/kafka"
CLUSTER_ID=$(cat /var/kafka/cluster_id)

echo "=== Phase 1: Génération des métadonnées KRaft ==="

# Créer le répertoire des métadonnées KRaft
mkdir -p /var/kafka/kraft-combined-logs

# Formater le stockage KRaft
$KAFKA_HOME/bin/kafka-storage.sh format \
    --config $KAFKA_HOME/config/kraft/server.properties \
    --cluster-id $CLUSTER_ID \
    --ignore-formatted

echo "=== Phase 2: Migration des métadonnées depuis ZooKeeper ==="

# Exporter les métadonnées ZooKeeper
$KAFKA_HOME/bin/kafka-metadata.sh snapshot \
    --snapshot /tmp/kraft-snapshot \
    --cluster-id $CLUSTER_ID

# Vérifier l'intégrité de la migration
$KAFKA_HOME/bin/kafka-metadata.sh verify \
    --snapshot /tmp/kraft-snapshot \
    --cluster-id $CLUSTER_ID

echo "=== Phase 3: Démarrage en mode KRaft ==="

# Les brokers doivent être redémarrés avec la nouvelle configuration
# Ceci est géré par le rolling restart orchestré

echo "Migration préparée. Exécuter le rolling restart avec la nouvelle configuration."
```

> **Note de terrain**
>
> *Contexte* : Migration KRaft d'un cluster Kafka de 15 brokers dans une entreprise de commerce électronique.
>
> *Défi* : Le cluster gérait 50 000 topics avec des centaines de milliers de partitions. La migration devait s'effectuer sans interruption de service pendant la période des fêtes.
>
> *Solution* : Migration en trois phases étalées sur 6 semaines avec fenêtres de maintenance nocturnes. Déploiement de contrôleurs KRaft dédiés avant la migration pour valider la stabilité.
>
> *Leçon* : Prévoir un temps de coexistence ZooKeeper/KRaft plus long que prévu initialement. Les comportements subtils ne se révèlent qu'en production avec charge réelle.

### III.11.1.3 Rolling Upgrades et Zero-Downtime

Les mises à niveau sans interruption exploitent la réplication Kafka pour maintenir la disponibilité pendant le processus.

**Prérequis pour le rolling upgrade**

```yaml
# Vérifications pré-upgrade

prerequisites:
  cluster_health:
    - all_brokers_online: true
    - under_replicated_partitions: 0
    - offline_partitions: 0
    - isr_shrink_rate: 0
  
  configuration:
    - min_insync_replicas: ≥ 2
    - replication_factor: ≥ 3
    - unclean_leader_election: false
  
  capacity:
    - disk_usage: < 70%
    - cpu_headroom: > 30%
    - network_headroom: > 30%
```

**Procédure de rolling upgrade**

```bash
#!/bin/bash
# scripts/rolling_upgrade.sh

BROKERS="kafka-1 kafka-2 kafka-3 kafka-4 kafka-5"
NEW_VERSION="3.7.0"
KAFKA_HOME="/opt/kafka"

wait_for_isr_sync() {
    local broker_id=$1
    echo "Attente de la synchronisation ISR pour broker $broker_id..."
  
    while true; do
        under_replicated=$($KAFKA_HOME/bin/kafka-topics.sh \
            --bootstrap-server localhost:9092 \
            --describe \
            --under-replicated-partitions | wc -l)
    
        if [ "$under_replicated" -eq 0 ]; then
            echo "ISR synchronisé pour toutes les partitions"
            break
        fi
    
        echo "Partitions sous-répliquées: $under_replicated. Attente..."
        sleep 30
    done
}

controlled_shutdown() {
    local broker=$1
    echo "Arrêt contrôlé de $broker..."
  
    ssh $broker "sudo systemctl stop kafka"
  
    # Attendre que le broker soit complètement arrêté
    sleep 10
}

upgrade_broker() {
    local broker=$1
    echo "Mise à niveau de $broker vers $NEW_VERSION..."
  
    ssh $broker << 'EOF'
        # Backup de la configuration
        cp -r /opt/kafka/config /opt/kafka/config.backup
    
        # Téléchargement et installation de la nouvelle version
        wget -q "https://downloads.apache.org/kafka/$NEW_VERSION/kafka_2.13-$NEW_VERSION.tgz" \
            -O /tmp/kafka.tgz
    
        tar -xzf /tmp/kafka.tgz -C /opt/
        rm -rf /opt/kafka
        mv /opt/kafka_2.13-$NEW_VERSION /opt/kafka
    
        # Restauration de la configuration
        cp -r /opt/kafka/config.backup/* /opt/kafka/config/
    
        # Ajustement du protocole inter-broker si nécessaire
        # (commenté - à activer selon la stratégie de migration)
        # echo "inter.broker.protocol.version=3.6" >> /opt/kafka/config/server.properties
EOF
}

start_broker() {
    local broker=$1
    echo "Démarrage de $broker..."
  
    ssh $broker "sudo systemctl start kafka"
  
    # Attendre que le broker rejoigne le cluster
    sleep 30
  
    # Vérifier que le broker est en ligne
    local broker_id=$(ssh $broker "cat /var/kafka/meta.properties | grep broker.id | cut -d= -f2")
  
    while true; do
        online=$($KAFKA_HOME/bin/kafka-broker-api-versions.sh \
            --bootstrap-server $broker:9092 2>/dev/null | grep -c "ApiVersion")
    
        if [ "$online" -gt 0 ]; then
            echo "Broker $broker_id en ligne"
            break
        fi
    
        echo "Attente du démarrage de $broker..."
        sleep 10
    done
}

# Boucle principale de mise à niveau
for broker in $BROKERS; do
    echo "=========================================="
    echo "Traitement de $broker"
    echo "=========================================="
  
    # 1. Arrêt contrôlé
    controlled_shutdown $broker
  
    # 2. Attendre la réélection des leaders
    sleep 60
  
    # 3. Mise à niveau
    upgrade_broker $broker
  
    # 4. Redémarrage
    start_broker $broker
  
    # 5. Attendre la synchronisation complète
    wait_for_isr_sync $broker
  
    echo "$broker mis à niveau avec succès"
    echo ""
done

echo "Rolling upgrade terminé"
```

### III.11.1.4 Gestion des Protocoles et Compatibilité

La compatibilité entre versions Kafka repose sur deux paramètres critiques : `inter.broker.protocol.version` et `log.message.format.version`.

**Stratégie de migration des protocoles**

```
Étape 1: Mise à niveau des binaires (tous les brokers)
         inter.broker.protocol.version = ancienne version
         log.message.format.version = ancienne version
     
Étape 2: Mise à niveau du protocole inter-broker
         inter.broker.protocol.version = nouvelle version
         log.message.format.version = ancienne version
         (Rolling restart)
     
Étape 3: Mise à niveau du format des messages
         inter.broker.protocol.version = nouvelle version
         log.message.format.version = nouvelle version
         (Rolling restart)
```

> **Anti-patron**
>
> Mettre à niveau simultanément les binaires et les protocoles. Cette approche empêche le rollback en cas de problème car les anciens brokers ne peuvent plus communiquer avec les nouveaux. Toujours procéder en phases distinctes avec validation intermédiaire.

---

## III.11.2 Mobilité des Données

### III.11.2.1 Réplication Inter-Clusters avec MirrorMaker 2

MirrorMaker 2 (MM2) assure la réplication des données entre clusters Kafka, permettant des architectures multi-région et des stratégies de reprise après sinistre.

**Architectures de réplication**

| Architecture   | Description                                           | Cas d'usage           |
| -------------- | ----------------------------------------------------- | --------------------- |
| Active-Passive | Un cluster primaire, un réplica en lecture seule     | Disaster Recovery     |
| Active-Active  | Deux clusters acceptant les écritures                | Géo-distribution     |
| Hub-and-Spoke  | Cluster central agrégant plusieurs régions          | Analytics centralisé |
| Mesh           | Réplication bidirectionnelle entre tous les clusters | Fédération globale  |

**Configuration MirrorMaker 2 pour Active-Passive**

```properties
# mm2-config.properties - Configuration Active-Passive

# Définition des clusters
clusters = primary, dr

primary.bootstrap.servers = kafka-primary-1:9092,kafka-primary-2:9092,kafka-primary-3:9092
dr.bootstrap.servers = kafka-dr-1:9092,kafka-dr-2:9092,kafka-dr-3:9092

# Réplication Primary -> DR
primary->dr.enabled = true
primary->dr.topics = .*
primary->dr.topics.blacklist = .*\.internal, __.*

# Configuration du connecteur source
primary->dr.source.cluster.bootstrap.servers = kafka-primary-1:9092,kafka-primary-2:9092
primary->dr.source.cluster.security.protocol = SASL_SSL
primary->dr.source.cluster.sasl.mechanism = SCRAM-SHA-512

# Configuration du connecteur sink
primary->dr.target.cluster.bootstrap.servers = kafka-dr-1:9092,kafka-dr-2:9092
primary->dr.target.cluster.security.protocol = SASL_SSL

# Synchronisation des offsets
primary->dr.sync.group.offsets.enabled = true
primary->dr.sync.group.offsets.interval.seconds = 10

# Synchronisation des ACL
primary->dr.acl.sync.enabled = true

# Réplication des topics de configuration
primary->dr.config.sync.enabled = true

# Performance
primary->dr.replication.factor = 3
primary->dr.offset.lag.max = 100
primary->dr.producer.batch.size = 16384
primary->dr.producer.linger.ms = 5

# Métriques
primary->dr.emit.heartbeats.enabled = true
primary->dr.emit.checkpoints.enabled = true
primary->dr.emit.heartbeats.interval.seconds = 5
```

**Déploiement MirrorMaker 2 sur Kubernetes**

```yaml
# mirrormaker2-deployment.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: mirrormaker2
  namespace: kafka
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mirrormaker2
  template:
    metadata:
      labels:
        app: mirrormaker2
    spec:
      containers:
        - name: mirrormaker2
          image: confluentinc/cp-kafka:7.5.0
          command:
            - /bin/bash
            - -c
            - |
              connect-mirror-maker /etc/mm2/mm2-config.properties
          resources:
            requests:
              memory: "2Gi"
              cpu: "1000m"
            limits:
              memory: "4Gi"
              cpu: "2000m"
          volumeMounts:
            - name: mm2-config
              mountPath: /etc/mm2
            - name: secrets
              mountPath: /etc/kafka/secrets
          env:
            - name: KAFKA_HEAP_OPTS
              value: "-Xms1g -Xmx2g"
            - name: KAFKA_JMX_PORT
              value: "9999"
          ports:
            - containerPort: 8083
              name: rest
            - containerPort: 9999
              name: jmx
          livenessProbe:
            httpGet:
              path: /connectors
              port: 8083
            initialDelaySeconds: 60
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /connectors
              port: 8083
            initialDelaySeconds: 30
            periodSeconds: 10
      volumes:
        - name: mm2-config
          configMap:
            name: mm2-config
        - name: secrets
          secret:
            secretName: kafka-credentials
```

### III.11.2.2 Gestion du Lag de Réplication

Le lag de réplication représente le retard entre les données du cluster source et leur réplique. Sa surveillance est critique pour les architectures de reprise après sinistre.

**Métriques de lag à surveiller**

```java
// Monitoring du lag MirrorMaker 2

public class MirrorMakerLagMonitor {
  
    private final AdminClient sourceAdmin;
    private final AdminClient targetAdmin;
    private final MeterRegistry meterRegistry;
  
    public void measureReplicationLag() {
        // Récupérer les offsets source
        Map<TopicPartition, Long> sourceOffsets = getLatestOffsets(sourceAdmin);
    
        // Récupérer les offsets répliqués
        Map<TopicPartition, Long> targetOffsets = getLatestOffsets(targetAdmin);
    
        // Calculer le lag par partition
        sourceOffsets.forEach((tp, sourceOffset) -> {
            String targetTopic = "primary." + tp.topic();
            TopicPartition targetTp = new TopicPartition(targetTopic, tp.partition());
        
            Long targetOffset = targetOffsets.getOrDefault(targetTp, 0L);
            long lag = sourceOffset - targetOffset;
        
            // Enregistrer la métrique
            Gauge.builder("mm2.replication.lag")
                .tag("source.topic", tp.topic())
                .tag("partition", String.valueOf(tp.partition()))
                .register(meterRegistry)
                .set(lag);
        
            // Alerter si le lag dépasse le seuil
            if (lag > LAG_THRESHOLD) {
                alertOnHighLag(tp, lag);
            }
        });
    }
  
    public Duration estimateRecoveryTime(String topic) {
        // Mesurer le débit de réplication
        double replicationRate = measureReplicationRate(topic);
    
        // Calculer le lag total
        long totalLag = calculateTotalLag(topic);
    
        // Estimer le temps de rattrapage
        if (replicationRate > 0) {
            return Duration.ofSeconds((long) (totalLag / replicationRate));
        }
        return Duration.ofDays(365); // Valeur sentinelle si pas de réplication
    }
}
```

**Alerting sur le lag de réplication**

```yaml
# prometheus-rules.yaml

groups:
  - name: mirrormaker2_alerts
    rules:
      - alert: MM2ReplicationLagHigh
        expr: mm2_replication_lag > 10000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Lag de réplication élevé"
          description: "Le lag de réplication pour {{ $labels.source_topic }} partition {{ $labels.partition }} est de {{ $value }} messages"
      
      - alert: MM2ReplicationLagCritical
        expr: mm2_replication_lag > 100000
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Lag de réplication critique"
          description: "Risque de perte de données en cas de basculement. Lag: {{ $value }}"
      
      - alert: MM2ReplicationStopped
        expr: rate(mm2_records_replicated_total[5m]) == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Réplication MirrorMaker 2 arrêtée"
          description: "Aucun message répliqué depuis 5 minutes"
```

### III.11.2.3 Migration de Topics entre Clusters

La migration de topics d'un cluster à un autre requiert une planification minutieuse pour préserver l'intégrité des données et minimiser l'interruption.

**Stratégies de migration**

| Stratégie        | Description                           | Downtime           | Complexité                  |
| ----------------- | ------------------------------------- | ------------------ | ---------------------------- |
| Dual-Write        | Écriture simultanée source et cible | Aucun              | Haute (coordination clients) |
| MirrorMaker       | Réplication puis basculement         | Minimal (secondes) | Moyenne                      |
| Export/Import     | Sauvegarde puis restauration          | Élevé            | Faible                       |
| Consumer-Producer | Application de migration dédiée     | Variable           | Moyenne                      |

**Procédure de migration avec MirrorMaker**

```bash
#!/bin/bash
# scripts/migrate_topic.sh

SOURCE_CLUSTER="kafka-old:9092"
TARGET_CLUSTER="kafka-new:9092"
TOPIC_TO_MIGRATE="orders.created"
CONSUMER_GROUPS="order-processor,analytics-consumer"

echo "=== Phase 1: Démarrage de la réplication ==="

# Configurer et démarrer MirrorMaker pour le topic spécifique
cat > /tmp/mm2-migration.properties << EOF
clusters = source, target
source.bootstrap.servers = $SOURCE_CLUSTER
target.bootstrap.servers = $TARGET_CLUSTER
source->target.enabled = true
source->target.topics = $TOPIC_TO_MIGRATE
source->target.sync.group.offsets.enabled = true
EOF

# Démarrer MirrorMaker
connect-mirror-maker /tmp/mm2-migration.properties &
MM2_PID=$!

echo "MirrorMaker démarré (PID: $MM2_PID)"

echo "=== Phase 2: Attente de la synchronisation ==="

while true; do
    # Vérifier le lag
    lag=$(kafka-consumer-groups.sh --bootstrap-server $TARGET_CLUSTER \
        --describe --group mm2-source \
        | grep $TOPIC_TO_MIGRATE \
        | awk '{sum += $6} END {print sum}')
  
    if [ "$lag" -lt 100 ]; then
        echo "Lag acceptable: $lag messages"
        break
    fi
  
    echo "Lag actuel: $lag messages. Attente..."
    sleep 10
done

echo "=== Phase 3: Basculement des consommateurs ==="

for group in $(echo $CONSUMER_GROUPS | tr ',' ' '); do
    echo "Migration du groupe $group..."
  
    # Récupérer les offsets depuis le topic checkpoint
    kafka-consumer-groups.sh --bootstrap-server $TARGET_CLUSTER \
        --group $group \
        --reset-offsets \
        --topic source.$TOPIC_TO_MIGRATE \
        --to-earliest \
        --execute
done

echo "=== Phase 4: Validation ==="

# Vérifier que les consommateurs lisent depuis le nouveau cluster
for group in $(echo $CONSUMER_GROUPS | tr ',' ' '); do
    kafka-consumer-groups.sh --bootstrap-server $TARGET_CLUSTER \
        --describe --group $group
done

echo "=== Phase 5: Nettoyage ==="

# Arrêter MirrorMaker
kill $MM2_PID

echo "Migration terminée. Vérifier le fonctionnement avant de supprimer l'ancien topic."
```

> **Note de terrain**
>
> *Contexte* : Migration de 200 topics d'un cluster Kafka on-premise vers Confluent Cloud pour une entreprise de logistique.
>
> *Défi* : Les applications ne pouvaient tolérer plus de 30 secondes d'interruption pendant les heures d'affaires.
>
> *Solution* : Migration topic par topic sur 3 mois avec dual-write temporaire pour les topics critiques. Automatisation complète du processus avec rollback automatique si le lag dépassait 1000 messages.
>
> *Leçon* : Sous-estimer le temps de migration est l'erreur la plus courante. Prévoir 50 % de marge sur les estimations initiales.

---

## III.11.3 Surveillance du Cluster Kafka

### III.11.3.1 Architecture d'Observabilité

Une surveillance efficace de Kafka repose sur trois piliers : métriques, logs et traces. L'architecture d'observabilité doit couvrir l'ensemble de la chaîne, des brokers aux applications clientes.

**Stack d'observabilité recommandée**

```
┌─────────────────────────────────────────────────────────────────┐
│                      Visualisation                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Grafana   │  │   Kibana    │  │    Jaeger   │              │
│  │  (Metrics)  │  │   (Logs)    │  │  (Traces)   │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                      │
├─────────┼────────────────┼────────────────┼─────────────────────┤
│         │    Stockage    │                │                      │
│  ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐              │
│  │ Prometheus  │  │Elasticsearch│  │    Tempo    │              │
│  │   / Mimir   │  │             │  │             │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                      │
├─────────┼────────────────┼────────────────┼─────────────────────┤
│         │   Collection   │                │                      │
│  ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐              │
│  │JMX Exporter │  │  Filebeat   │  │ OTel Agent  │              │
│  │             │  │  / Fluentd  │  │             │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                      │
├─────────┴────────────────┴────────────────┴─────────────────────┤
│                    Sources de données                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Brokers   │  │   Clients   │  │   Connect   │              │
│  │    Kafka    │  │  Producers  │  │   Workers   │              │
│  │             │  │  Consumers  │  │             │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

### III.11.3.2 Métriques Essentielles des Brokers

**Métriques de santé du cluster**

```yaml
# prometheus-kafka-rules.yaml

groups:
  - name: kafka_broker_health
    rules:
      # Partitions sous-répliquées
      - record: kafka:under_replicated_partitions:sum
        expr: sum(kafka_server_replicamanager_underreplicatedpartitions)
    
      # Partitions hors ligne
      - record: kafka:offline_partitions:sum
        expr: sum(kafka_controller_kafkacontroller_offlinepartitionscount)
    
      # Élections de leader non propres
      - record: kafka:unclean_leader_elections:rate5m
        expr: sum(rate(kafka_controller_controllerstats_uncleanleaderelectionspersec[5m]))
    
      # ISR en contraction
      - record: kafka:isr_shrinks:rate5m
        expr: sum(rate(kafka_server_replicamanager_isrshrinkspersec[5m]))
```

**Tableau de bord Grafana - Santé du cluster**

```json
{
  "panels": [
    {
      "title": "Partitions sous-répliquées",
      "type": "stat",
      "targets": [
        {
          "expr": "sum(kafka_server_replicamanager_underreplicatedpartitions)",
          "legendFormat": "Under-replicated"
        }
      ],
      "thresholds": {
        "mode": "absolute",
        "steps": [
          {"color": "green", "value": 0},
          {"color": "yellow", "value": 1},
          {"color": "red", "value": 10}
        ]
      }
    },
    {
      "title": "Débit d'entrée par broker",
      "type": "graph",
      "targets": [
        {
          "expr": "sum(rate(kafka_server_brokertopicmetrics_bytesinpersec[5m])) by (instance)",
          "legendFormat": "{{ instance }}"
        }
      ]
    },
    {
      "title": "Latence de réplication",
      "type": "graph",
      "targets": [
        {
          "expr": "kafka_server_replicafetchermanager_maxlag",
          "legendFormat": "Max Lag"
        }
      ]
    }
  ]
}
```

**Métriques de performance des brokers**

| Métrique                            | Description                         | Seuil d'alerte          |
| ------------------------------------ | ----------------------------------- | ----------------------- |
| `RequestHandlerAvgIdlePercent`     | Utilisation des threads de requête | < 30%                   |
| `NetworkProcessorAvgIdlePercent`   | Utilisation des threads réseau     | < 30%                   |
| `UnderReplicatedPartitions`        | Partitions sans réplicas complets  | > 0                     |
| `ActiveControllerCount`            | Nombre de contrôleurs actifs       | ≠ 1                    |
| `OfflinePartitionsCount`           | Partitions sans leader              | > 0                     |
| `BytesInPerSec`/`BytesOutPerSec` | Débit réseau                      | Dépend de la capacité |
| `TotalProduceRequestsPerSec`       | Requêtes de production             | Baseline + 50%          |
| `TotalFetchRequestsPerSec`         | Requêtes de consommation           | Baseline + 50%          |

### III.11.3.3 Métriques des Producers et Consumers

**Configuration JMX pour les clients**

```java
// Configuration des métriques producer

public class MonitoredKafkaProducer<K, V> {
  
    private final KafkaProducer<K, V> producer;
    private final MeterRegistry meterRegistry;
  
    public MonitoredKafkaProducer(Properties props, MeterRegistry registry) {
        this.meterRegistry = registry;
        this.producer = new KafkaProducer<>(props);
    
        // Enregistrer les métriques
        registerMetrics();
    }
  
    private void registerMetrics() {
        // Métriques du producer
        producer.metrics().forEach((name, metric) -> {
            if (isImportantMetric(name)) {
                Gauge.builder("kafka.producer." + name.name())
                    .tag("client.id", metric.metricName().tags().get("client-id"))
                    .register(meterRegistry);
            }
        });
    }
  
    private boolean isImportantMetric(MetricName name) {
        return name.name().contains("record-send-rate") ||
               name.name().contains("record-error-rate") ||
               name.name().contains("request-latency-avg") ||
               name.name().contains("batch-size-avg") ||
               name.name().contains("buffer-available-bytes");
    }
}
```

**Métriques critiques des consumers**

```yaml
# Règles d'alerte pour les consumers

groups:
  - name: kafka_consumer_alerts
    rules:
      - alert: ConsumerLagHigh
        expr: |
          sum(kafka_consumer_group_lag) by (group, topic) > 10000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Lag élevé pour le groupe {{ $labels.group }}"
          description: "Le consumer group {{ $labels.group }} a un lag de {{ $value }} sur {{ $labels.topic }}"
      
      - alert: ConsumerLagGrowing
        expr: |
          deriv(kafka_consumer_group_lag[10m]) > 100
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Lag en croissance pour {{ $labels.group }}"
          description: "Le lag augmente de {{ $value }} messages/seconde"
      
      - alert: ConsumerGroupInactive
        expr: |
          kafka_consumer_group_members == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Consumer group inactif"
          description: "Le groupe {{ $labels.group }} n'a aucun membre actif"
```

### III.11.3.4 Surveillance des Topics et Partitions

**Script de diagnostic des topics**

```bash
#!/bin/bash
# scripts/topic_health_check.sh

BOOTSTRAP_SERVERS="${1:-localhost:9092}"

echo "=== Diagnostic de santé des topics ==="
echo ""

# Partitions sous-répliquées
echo "--- Partitions sous-répliquées ---"
kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVERS \
    --describe \
    --under-replicated-partitions

# Partitions hors ligne
echo ""
echo "--- Partitions hors ligne ---"
kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVERS \
    --describe \
    --unavailable-partitions

# Topics avec ISR réduit
echo ""
echo "--- Topics avec ISR < replication.factor ---"
kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVERS \
    --describe \
    | awk '/Isr:/ {
        split($0, a, "Replicas:");
        split(a[2], b, "Isr:");
        replicas = gsub(/,/, ",", b[1]) + 1;
        isr = gsub(/,/, ",", b[2]) + 1;
        if (isr < replicas) print $0
    }'

# Distribution des partitions par broker
echo ""
echo "--- Distribution des leaders par broker ---"
kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVERS \
    --describe \
    | grep "Leader:" \
    | awk '{print $6}' \
    | sort \
    | uniq -c \
    | sort -rn

# Topics les plus volumineux
echo ""
echo "--- Top 10 topics par nombre de partitions ---"
kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVERS \
    --describe \
    | grep "PartitionCount:" \
    | awk '{print $4, $2}' \
    | sort -rn \
    | head -10
```

### III.11.3.5 Centralisation des Logs

**Configuration Filebeat pour Kafka**

```yaml
# filebeat-kafka.yml

filebeat.inputs:
  - type: log
    enabled: true
    paths:
      - /var/log/kafka/server.log
      - /var/log/kafka/controller.log
      - /var/log/kafka/state-change.log
    multiline:
      pattern: '^\['
      negate: true
      match: after
    fields:
      log_type: kafka-broker
      environment: production
    fields_under_root: true

  - type: log
    enabled: true
    paths:
      - /var/log/kafka/kafka-request.log
    fields:
      log_type: kafka-request
    fields_under_root: true

processors:
  - dissect:
      tokenizer: "[%{timestamp}] %{level} %{message}"
      field: "message"
      target_prefix: "kafka"
      when:
        equals:
          log_type: kafka-broker
      
  - add_host_metadata:
      when.not.contains.tags: forwarded
  
  - add_cloud_metadata: ~

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "kafka-logs-%{+yyyy.MM.dd}"
  
setup.template:
  name: "kafka-logs"
  pattern: "kafka-logs-*"
  settings:
    index.number_of_shards: 3
    index.number_of_replicas: 1
```

**Patterns de logs à surveiller**

```yaml
# Patterns critiques dans les logs Kafka

alert_patterns:
  critical:
    - pattern: "FATAL"
      description: "Erreur fatale du broker"
      action: "page_on_call"
  
    - pattern: "OutOfMemoryError"
      description: "Dépassement mémoire"
      action: "page_on_call"
  
    - pattern: "No space left on device"
      description: "Disque plein"
      action: "page_on_call"
  
  warning:
    - pattern: "WARN.*ISR.*shrunk"
      description: "ISR en contraction"
      action: "create_ticket"
  
    - pattern: "WARN.*Unable to connect"
      description: "Problème de connectivité"
      action: "investigate"
  
    - pattern: "WARN.*Request.*timed out"
      description: "Timeout de requête"
      action: "investigate"
  
  info:
    - pattern: "INFO.*Leader.*changed"
      description: "Changement de leader"
      action: "log_only"
  
    - pattern: "INFO.*Partition.*reassignment"
      description: "Réassignation de partition"
      action: "log_only"
```

---

## III.11.4 Clinique d'Optimisation des Performances

### III.11.4.1 Diagnostic des Problèmes de Performance

L'optimisation des performances Kafka suit une méthodologie systématique qui identifie les goulots d'étranglement avant d'appliquer des corrections.

**Arbre de décision du diagnostic**

```
Performance dégradée
│
├── Latence élevée ?
│   ├── Latence producer ?
│   │   ├── Batch size trop petit → Augmenter batch.size
│   │   ├── Linger.ms trop bas → Augmenter linger.ms
│   │   └── Acks=all avec ISR lent → Vérifier santé réplicas
│   │
│   └── Latence consumer ?
│       ├── Traitement lent → Optimiser logique applicative
│       ├── Rebalancing fréquent → Augmenter session.timeout.ms
│       └── Fetch size inadapté → Ajuster fetch.min.bytes
│
├── Débit insuffisant ?
│   ├── Côté broker ?
│   │   ├── CPU saturé → Ajouter brokers ou partitions
│   │   ├── Disque saturé → Vérifier I/O, ajouter disques
│   │   └── Réseau saturé → Vérifier bande passante
│   │
│   ├── Côté producer ?
│   │   ├── Sérialisation lente → Optimiser sérialisation
│   │   └── Compression inefficace → Changer algorithme
│   │
│   └── Côté consumer ?
│       ├── Pas assez de partitions → Augmenter partitions
│       └── Pas assez de consumers → Ajouter instances
│
└── Instabilité (erreurs intermittentes) ?
    ├── Timeout fréquents → Ajuster timeouts
    ├── Rebalancing constant → Vérifier heartbeat
    └── Déconnexions → Vérifier réseau et configuration
```

### III.11.4.2 Optimisation des Producers

**Paramètres critiques du producer**

```java
// Configuration producer optimisée pour le débit

public Properties createHighThroughputProducerConfig() {
    Properties props = new Properties();
  
    // Identification
    props.put(ProducerConfig.CLIENT_ID_CONFIG, "high-throughput-producer");
    props.put(ProducerConfig.BOOTSTRAP_SERVERS_CONFIG, bootstrapServers);
  
    // Sérialisation
    props.put(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class);
    props.put(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, KafkaAvroSerializer.class);
  
    // Batching - clé de la performance
    props.put(ProducerConfig.BATCH_SIZE_CONFIG, 65536);        // 64 KB par batch
    props.put(ProducerConfig.LINGER_MS_CONFIG, 10);            // Attendre 10ms max
    props.put(ProducerConfig.BUFFER_MEMORY_CONFIG, 67108864);  // 64 MB de buffer
  
    // Compression - réduire bande passante et stockage
    props.put(ProducerConfig.COMPRESSION_TYPE_CONFIG, "lz4"); // Bon compromis vitesse/ratio
  
    // Fiabilité - ajuster selon besoins
    props.put(ProducerConfig.ACKS_CONFIG, "all");             // Durabilité maximale
    props.put(ProducerConfig.RETRIES_CONFIG, 3);
    props.put(ProducerConfig.RETRY_BACKOFF_MS_CONFIG, 100);
  
    // Idempotence - éviter les doublons
    props.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, true);
  
    // Timeouts
    props.put(ProducerConfig.REQUEST_TIMEOUT_MS_CONFIG, 30000);
    props.put(ProducerConfig.DELIVERY_TIMEOUT_MS_CONFIG, 120000);
  
    // Performance réseau
    props.put(ProducerConfig.SEND_BUFFER_CONFIG, 131072);     // 128 KB
    props.put(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, 5);
  
    return props;
}
```

**Comparaison des algorithmes de compression**

| Algorithme | Ratio compression | Vitesse compression | Vitesse décompression | Recommandation                  |
| ---------- | ----------------- | ------------------- | ---------------------- | ------------------------------- |
| none       | 1:1               | N/A                 | N/A                    | Tests uniquement                |
| gzip       | ~70%              | Lente               | Moyenne                | Stockage long terme             |
| snappy     | ~50%              | Rapide              | Très rapide           | Legacy, compatibilité          |
| lz4        | ~55%              | Très rapide        | Très rapide           | **Production générale** |
| zstd       | ~65%              | Rapide              | Rapide                 | Meilleur ratio moderne          |

> **Note de terrain**
>
> *Contexte* : Optimisation d'un pipeline d'ingestion IoT traitant 500 000 messages/seconde.
>
> *Défi* : Le débit plafonnait à 200 000 messages/seconde malgré des brokers non saturés.
>
> *Solution* : Augmentation du batch.size de 16 KB à 128 KB, linger.ms de 0 à 20 ms, et passage de snappy à lz4. Le débit a triplé.
>
> *Leçon* : Le batching est le levier d'optimisation le plus puissant. Un batch trop petit multiplie les round-trips réseau inutilement.

### III.11.4.3 Optimisation des Consumers

**Configuration consumer pour différents profils**

```java
// Consumer optimisé pour la latence
public Properties createLowLatencyConsumerConfig() {
    Properties props = new Properties();
  
    props.put(ConsumerConfig.FETCH_MIN_BYTES_CONFIG, 1);          // Pas d'attente
    props.put(ConsumerConfig.FETCH_MAX_WAIT_MS_CONFIG, 100);      // 100ms max
    props.put(ConsumerConfig.MAX_POLL_RECORDS_CONFIG, 100);       // Petits lots
    props.put(ConsumerConfig.SESSION_TIMEOUT_MS_CONFIG, 10000);
    props.put(ConsumerConfig.HEARTBEAT_INTERVAL_MS_CONFIG, 3000);
  
    return props;
}

// Consumer optimisé pour le débit
public Properties createHighThroughputConsumerConfig() {
    Properties props = new Properties();
  
    props.put(ConsumerConfig.FETCH_MIN_BYTES_CONFIG, 65536);      // 64 KB minimum
    props.put(ConsumerConfig.FETCH_MAX_BYTES_CONFIG, 52428800);   // 50 MB max
    props.put(ConsumerConfig.FETCH_MAX_WAIT_MS_CONFIG, 500);      // Attendre pour batching
    props.put(ConsumerConfig.MAX_POLL_RECORDS_CONFIG, 1000);      // Gros lots
    props.put(ConsumerConfig.MAX_PARTITION_FETCH_BYTES_CONFIG, 10485760); // 10 MB
  
    return props;
}
```

**Diagnostic du consumer lag**

```bash
#!/bin/bash
# scripts/diagnose_consumer_lag.sh

BOOTSTRAP_SERVERS="$1"
CONSUMER_GROUP="$2"

echo "=== Diagnostic du lag pour $CONSUMER_GROUP ==="
echo ""

# Lag par partition
echo "--- Lag par partition ---"
kafka-consumer-groups.sh --bootstrap-server $BOOTSTRAP_SERVERS \
    --describe --group $CONSUMER_GROUP \
    | column -t

# Tendance du lag (nécessite métriques historiques)
echo ""
echo "--- Analyse ---"

lag_data=$(kafka-consumer-groups.sh --bootstrap-server $BOOTSTRAP_SERVERS \
    --describe --group $CONSUMER_GROUP \
    | tail -n +2)

total_lag=0
max_lag=0
partitions_with_lag=0

while IFS= read -r line; do
    lag=$(echo "$line" | awk '{print $6}')
    if [ "$lag" != "-" ] && [ -n "$lag" ]; then
        total_lag=$((total_lag + lag))
        if [ "$lag" -gt 0 ]; then
            partitions_with_lag=$((partitions_with_lag + 1))
        fi
        if [ "$lag" -gt "$max_lag" ]; then
            max_lag=$lag
        fi
    fi
done <<< "$lag_data"

echo "Lag total: $total_lag"
echo "Lag maximum: $max_lag"
echo "Partitions avec lag: $partitions_with_lag"

# Recommandations
echo ""
echo "--- Recommandations ---"

if [ "$total_lag" -gt 100000 ]; then
    echo "⚠️  Lag élevé: Considérer l'ajout de consumers ou l'optimisation du traitement"
fi

if [ "$max_lag" -gt 50000 ] && [ "$partitions_with_lag" -lt 3 ]; then
    echo "⚠️  Lag concentré sur peu de partitions: Possible hot partition"
fi
```

### III.11.4.4 Optimisation des Brokers

**Configuration broker pour hautes performances**

```properties
# server.properties - Configuration optimisée

# Threads de traitement
num.network.threads=8
num.io.threads=16
num.replica.fetchers=4

# Buffers socket
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600

# Logs
log.flush.interval.messages=10000
log.flush.interval.ms=1000

# Réplication
replica.fetch.min.bytes=1
replica.fetch.max.bytes=10485760
replica.fetch.wait.max.ms=500
replica.lag.time.max.ms=30000

# Segments
log.segment.bytes=1073741824
log.retention.check.interval.ms=300000

# Compression
compression.type=producer

# OS page cache
log.flush.scheduler.interval.ms=9223372036854775807

# Quotas (optionnel)
# quota.producer.default=10485760
# quota.consumer.default=10485760
```

**Tuning du système d'exploitation**

```bash
#!/bin/bash
# scripts/tune_os_for_kafka.sh

echo "=== Configuration OS pour Kafka ==="

# Augmenter les limites de fichiers ouverts
cat >> /etc/security/limits.conf << EOF
kafka soft nofile 128000
kafka hard nofile 128000
kafka soft nproc 65536
kafka hard nproc 65536
EOF

# Paramètres réseau
cat >> /etc/sysctl.conf << EOF
# Buffers réseau
net.core.wmem_default=131072
net.core.rmem_default=131072
net.core.wmem_max=2097152
net.core.rmem_max=2097152
net.ipv4.tcp_wmem=4096 65536 2048000
net.ipv4.tcp_rmem=4096 65536 2048000

# Connexions
net.core.netdev_max_backlog=50000
net.ipv4.tcp_max_syn_backlog=30000
net.ipv4.tcp_max_tw_buckets=2000000
net.ipv4.tcp_tw_reuse=1
net.ipv4.tcp_fin_timeout=10

# Virtual memory
vm.swappiness=1
vm.dirty_background_ratio=5
vm.dirty_ratio=60
EOF

sysctl -p

# Désactiver les huge pages transparentes (recommandé pour Kafka)
echo never > /sys/kernel/mm/transparent_hugepage/enabled
echo never > /sys/kernel/mm/transparent_hugepage/defrag

echo "Configuration OS appliquée. Redémarrer Kafka pour appliquer les limites."
```

### III.11.4.5 Gestion des Hot Partitions

Les hot partitions surviennent lorsqu'une partition reçoit disproportionnément plus de trafic que les autres, créant un goulot d'étranglement.

**Diagnostic des hot partitions**

```java
// Outil de diagnostic des hot partitions

public class HotPartitionDetector {
  
    private final AdminClient adminClient;
  
    public Map<TopicPartition, PartitionStats> analyzeDistribution(String topic) {
        Map<TopicPartition, PartitionStats> stats = new HashMap<>();
    
        // Récupérer les offsets de début et fin
        Map<TopicPartition, Long> beginningOffsets = getBeginningOffsets(topic);
        Map<TopicPartition, Long> endOffsets = getEndOffsets(topic);
    
        // Calculer le volume par partition
        long totalMessages = 0;
        List<Long> volumes = new ArrayList<>();
    
        for (TopicPartition tp : endOffsets.keySet()) {
            long volume = endOffsets.get(tp) - beginningOffsets.get(tp);
            volumes.add(volume);
            totalMessages += volume;
        
            stats.put(tp, new PartitionStats(tp, volume));
        }
    
        // Calculer les statistiques
        double mean = (double) totalMessages / volumes.size();
        double stdDev = calculateStdDev(volumes, mean);
    
        // Identifier les outliers (> 2 écarts-types)
        for (PartitionStats ps : stats.values()) {
            double zScore = (ps.volume - mean) / stdDev;
            ps.setZScore(zScore);
        
            if (zScore > 2.0) {
                ps.setHotPartition(true);
                log.warn("Hot partition détectée: {} avec z-score {}", 
                    ps.partition, zScore);
            }
        }
    
        return stats;
    }
  
    public List<String> suggestRemediations(Map<TopicPartition, PartitionStats> stats) {
        List<String> suggestions = new ArrayList<>();
    
        long hotPartitionCount = stats.values().stream()
            .filter(PartitionStats::isHotPartition)
            .count();
    
        if (hotPartitionCount > 0) {
            suggestions.add("1. Vérifier la stratégie de partitionnement du producer");
            suggestions.add("2. Considérer une clé de partition plus distribuée");
            suggestions.add("3. Augmenter le nombre de partitions si la clé est nécessaire");
            suggestions.add("4. Implémenter un partitionneur personnalisé avec salage");
        }
    
        return suggestions;
    }
}
```

**Stratégies de résolution**

| Cause             | Symptôme                          | Solution                                 |
| ----------------- | ---------------------------------- | ---------------------------------------- |
| Clé constante    | Une partition monopolise le trafic | Ajouter un suffixe aléatoire à la clé |
| Distribution Zipf | Quelques clés dominent            | Partitionneur avec hachage modifié      |
| Pic temporel      | Partition hot par moments          | Pré-partitionnement temporel            |
| Mauvais hachage   | Distribution inégale              | Changer l'algorithme de hachage          |

---

## III.11.5 Reprise Après Sinistre et Basculement

### III.11.5.1 Stratégies de Continuité d'Activité

La stratégie de reprise après sinistre (Disaster Recovery, DR) définit comment l'organisation maintient ses opérations Kafka face à une défaillance majeure.

**Modèles de déploiement DR**

| Modèle        | RTO       | RPO      | Coût         | Complexité    |
| -------------- | --------- | -------- | ------------- | -------------- |
| Backup/Restore | Heures    | Heures   | Faible        | Faible         |
| Pilot Light    | 30-60 min | Minutes  | Moyen         | Moyenne        |
| Warm Standby   | 10-30 min | Secondes | Moyen-Élevé | Élevée       |
| Hot Standby    | < 5 min   | ~0       | Élevé       | Très élevée |
| Active-Active  | ~0        | ~0       | Très élevé | Maximale       |

**Architecture Hot Standby avec MirrorMaker 2**

```
┌─────────────────────────────────────────────────────────────────┐
│                      Région Primaire                            │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Cluster Kafka Principal                     │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐    │   │
│  │  │Broker 1 │  │Broker 2 │  │Broker 3 │  │Broker 4 │    │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              │ Réplication MM2                  │
│                              ▼                                  │
└──────────────────────────────┼──────────────────────────────────┘
                               │
                               │ Réseau inter-région
                               │
┌──────────────────────────────┼──────────────────────────────────┐
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Cluster Kafka DR (Standby)                  │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐    │   │
│  │  │Broker 1 │  │Broker 2 │  │Broker 3 │  │Broker 4 │    │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                      Région DR                                  │
└─────────────────────────────────────────────────────────────────┘
```

### III.11.5.2 Procédure de Basculement

**Runbook de basculement automatisé**

```bash
#!/bin/bash
# scripts/failover_to_dr.sh

set -e

PRIMARY_CLUSTER="kafka-primary.example.com:9092"
DR_CLUSTER="kafka-dr.example.com:9092"
MM2_CONNECTOR="mirrormaker2"
DNS_ZONE="example.com"
KAFKA_CNAME="kafka.example.com"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

check_primary_health() {
    log "Vérification de la santé du cluster primaire..."
  
    if kafka-broker-api-versions.sh --bootstrap-server $PRIMARY_CLUSTER \
        --timeout 10000 2>/dev/null; then
        return 0
    fi
    return 1
}

check_dr_health() {
    log "Vérification de la santé du cluster DR..."
  
    if ! kafka-broker-api-versions.sh --bootstrap-server $DR_CLUSTER \
        --timeout 10000 2>/dev/null; then
        log "ERREUR: Cluster DR non disponible"
        exit 1
    fi
    log "Cluster DR opérationnel"
}

check_replication_lag() {
    log "Vérification du lag de réplication..."
  
    lag=$(curl -s "http://mm2-metrics:8080/metrics" \
        | grep 'mm2_replication_lag' \
        | awk '{sum += $2} END {print sum}')
  
    log "Lag total: $lag messages"
  
    if [ "$lag" -gt 10000 ]; then
        log "ATTENTION: Lag élevé. Risque de perte de données: ~$lag messages"
        read -p "Continuer le basculement? (yes/no) " confirm
        if [ "$confirm" != "yes" ]; then
            log "Basculement annulé"
            exit 1
        fi
    fi
}

stop_mirrormaker() {
    log "Arrêt de MirrorMaker 2..."
  
    # Arrêt gracieux pour finaliser les réplications en cours
    curl -X PUT "http://kafka-connect:8083/connectors/$MM2_CONNECTOR/pause"
    sleep 30
    curl -X DELETE "http://kafka-connect:8083/connectors/$MM2_CONNECTOR"
  
    log "MirrorMaker arrêté"
}

update_dns() {
    log "Mise à jour DNS vers le cluster DR..."
  
    # Exemple avec AWS Route 53
    aws route53 change-resource-record-sets \
        --hosted-zone-id $HOSTED_ZONE_ID \
        --change-batch '{
            "Changes": [{
                "Action": "UPSERT",
                "ResourceRecordSet": {
                    "Name": "'$KAFKA_CNAME'",
                    "Type": "CNAME",
                    "TTL": 60,
                    "ResourceRecords": [{"Value": "'$DR_CLUSTER'"}]
                }
            }]
        }'
  
    log "DNS mis à jour. TTL: 60 secondes"
}

notify_teams() {
    log "Notification des équipes..."
  
    # Slack notification
    curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"🚨 BASCULEMENT KAFKA: Cluster DR activé. Cluster primaire indisponible."}' \
        $SLACK_WEBHOOK_URL
  
    # PagerDuty incident
    curl -X POST "https://events.pagerduty.com/v2/enqueue" \
        -H "Content-Type: application/json" \
        -d '{
            "routing_key": "'$PAGERDUTY_KEY'",
            "event_action": "trigger",
            "payload": {
                "summary": "Kafka Failover to DR",
                "severity": "critical",
                "source": "kafka-failover-script"
            }
        }'
}

validate_failover() {
    log "Validation du basculement..."
  
    # Test de production
    echo "test-failover-$(date +%s)" | kafka-console-producer.sh \
        --bootstrap-server $DR_CLUSTER \
        --topic failover-test
  
    # Test de consommation
    timeout 10 kafka-console-consumer.sh \
        --bootstrap-server $DR_CLUSTER \
        --topic failover-test \
        --from-beginning \
        --max-messages 1
  
    log "Validation réussie"
}

# === EXECUTION PRINCIPALE ===

log "=== DÉBUT DE LA PROCÉDURE DE BASCULEMENT ==="

# Vérifier si le basculement est nécessaire
if check_primary_health; then
    log "Le cluster primaire est accessible. Basculement non nécessaire."
    read -p "Forcer le basculement? (yes/no) " force
    if [ "$force" != "yes" ]; then
        exit 0
    fi
fi

# Exécuter le basculement
check_dr_health
check_replication_lag
stop_mirrormaker
update_dns
notify_teams

# Attendre la propagation DNS
log "Attente de la propagation DNS (120 secondes)..."
sleep 120

validate_failover

log "=== BASCULEMENT TERMINÉ ==="
log "Actions post-basculement requises:"
log "1. Vérifier les applications clientes"
log "2. Monitorer les métriques du cluster DR"
log "3. Planifier le retour sur le cluster primaire"
```

### III.11.5.3 Procédure de Retour (Failback)

Le retour vers le cluster primaire après réparation requiert une planification minutieuse pour éviter la perte de données produites pendant l'incident.

**Séquence de failback**

```
Phase 1: Préparation
├── Valider la santé du cluster primaire
├── Estimer le volume de données à resynchroniser
├── Planifier la fenêtre de maintenance
└── Notifier les parties prenantes

Phase 2: Resynchronisation
├── Démarrer MirrorMaker DR → Primary
├── Attendre la convergence (lag → 0)
├── Valider l'intégrité des données
└── Préparer les consommateurs

Phase 3: Basculement
├── Arrêter les producers sur DR
├── Attendre le drainage complet
├── Mettre à jour le DNS
├── Redémarrer les producers sur Primary
└── Migrer les offsets consommateurs

Phase 4: Nettoyage
├── Arrêter MirrorMaker DR → Primary
├── Reconfigurer MirrorMaker Primary → DR
├── Valider le fonctionnement nominal
└── Documentation post-mortem
```

> **Anti-patron**
>
> Exécuter un failback précipité sans resynchronisation complète. Les données produites sur le cluster DR pendant l'incident doivent être répliquées vers le primaire avant de rediriger le trafic. Un failback hâtif peut entraîner une perte de données significative.

### III.11.5.4 Tests de Reprise Après Sinistre

Les tests réguliers de DR valident la capacité réelle de l'organisation à basculer en situation de crise.

**Programme de tests DR**

```yaml
# dr-test-schedule.yaml

dr_tests:
  - name: "Test de communication"
    frequency: weekly
    duration: 15min
    scope: "Vérification connectivité MM2"
    impact: none
    automation: full
  
  - name: "Test de basculement partiel"
    frequency: monthly
    duration: 2h
    scope: "Basculement d'un topic non-critique"
    impact: minimal
    automation: partial
  
  - name: "Test de basculement complet"
    frequency: quarterly
    duration: 4h
    scope: "Simulation de perte du datacenter primaire"
    impact: planned_outage
    automation: manual_validation
  
  - name: "Test de chaos"
    frequency: semi_annually
    duration: 8h
    scope: "Injection de pannes multiples"
    impact: controlled
    automation: none
```

**Script de test DR automatisé**

```bash
#!/bin/bash
# scripts/dr_test.sh

TEST_ID="dr-test-$(date +%Y%m%d-%H%M%S)"
TEST_TOPIC="dr-test-topic-$TEST_ID"
MESSAGE_COUNT=10000

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$TEST_ID] $1" | tee -a /var/log/dr-tests.log
}

setup_test() {
    log "Création du topic de test..."
  
    kafka-topics.sh --bootstrap-server $PRIMARY_CLUSTER \
        --create --topic $TEST_TOPIC \
        --partitions 6 --replication-factor 3
  
    log "Attente de la réplication vers DR..."
    sleep 60
  
    # Vérifier que le topic est répliqué
    if ! kafka-topics.sh --bootstrap-server $DR_CLUSTER \
        --list | grep -q "primary.$TEST_TOPIC"; then
        log "ERREUR: Topic non répliqué vers DR"
        exit 1
    fi
}

produce_test_data() {
    log "Production de $MESSAGE_COUNT messages de test..."
  
    seq 1 $MESSAGE_COUNT | kafka-console-producer.sh \
        --bootstrap-server $PRIMARY_CLUSTER \
        --topic $TEST_TOPIC
  
    log "Production terminée"
}

wait_for_replication() {
    log "Attente de la réplication complète..."
  
    while true; do
        primary_offset=$(kafka-run-class.sh kafka.tools.GetOffsetShell \
            --broker-list $PRIMARY_CLUSTER \
            --topic $TEST_TOPIC \
            | awk -F: '{sum += $3} END {print sum}')
    
        dr_offset=$(kafka-run-class.sh kafka.tools.GetOffsetShell \
            --broker-list $DR_CLUSTER \
            --topic "primary.$TEST_TOPIC" \
            | awk -F: '{sum += $3} END {print sum}')
    
        lag=$((primary_offset - dr_offset))
    
        log "Lag actuel: $lag messages"
    
        if [ "$lag" -eq 0 ]; then
            log "Réplication complète"
            break
        fi
    
        sleep 5
    done
}

verify_data_integrity() {
    log "Vérification de l'intégrité des données..."
  
    primary_checksum=$(kafka-console-consumer.sh \
        --bootstrap-server $PRIMARY_CLUSTER \
        --topic $TEST_TOPIC \
        --from-beginning --max-messages $MESSAGE_COUNT \
        --timeout-ms 30000 2>/dev/null | md5sum)
  
    dr_checksum=$(kafka-console-consumer.sh \
        --bootstrap-server $DR_CLUSTER \
        --topic "primary.$TEST_TOPIC" \
        --from-beginning --max-messages $MESSAGE_COUNT \
        --timeout-ms 30000 2>/dev/null | md5sum)
  
    if [ "$primary_checksum" == "$dr_checksum" ]; then
        log "✓ Intégrité des données validée"
        return 0
    else
        log "✗ ERREUR: Checksums différents"
        log "  Primary: $primary_checksum"
        log "  DR: $dr_checksum"
        return 1
    fi
}

cleanup() {
    log "Nettoyage..."
  
    kafka-topics.sh --bootstrap-server $PRIMARY_CLUSTER \
        --delete --topic $TEST_TOPIC
  
    # Le topic miroir sera supprimé automatiquement par MM2
}

measure_rpo() {
    log "Mesure du RPO..."
  
    # Produire un message avec timestamp
    start_time=$(date +%s%N)
    echo "rpo-test-$start_time" | kafka-console-producer.sh \
        --bootstrap-server $PRIMARY_CLUSTER \
        --topic $TEST_TOPIC
  
    # Attendre sa réplication
    while true; do
        if kafka-console-consumer.sh \
            --bootstrap-server $DR_CLUSTER \
            --topic "primary.$TEST_TOPIC" \
            --from-beginning --max-messages 1 \
            --timeout-ms 1000 2>/dev/null | grep -q "rpo-test-$start_time"; then
            end_time=$(date +%s%N)
            break
        fi
    done
  
    rpo_ms=$(( (end_time - start_time) / 1000000 ))
    log "RPO mesuré: ${rpo_ms}ms"
}

# === EXECUTION ===

log "=== DÉBUT DU TEST DR ==="

setup_test
produce_test_data
wait_for_replication
verify_data_integrity
result=$?

measure_rpo
cleanup

if [ $result -eq 0 ]; then
    log "=== TEST DR RÉUSSI ==="
else
    log "=== TEST DR ÉCHOUÉ ==="
    exit 1
fi
```

### III.11.5.5 Documentation et Runbooks

La documentation opérationnelle constitue un actif critique pour la gestion des incidents.

**Structure de runbook recommandée**

```markdown
# Runbook: [Nom de la procédure]

## Informations générales
- **Dernière mise à jour**: YYYY-MM-DD
- **Propriétaire**: [Équipe/Personne]
- **Temps estimé**: X minutes
- **Niveau d'expertise requis**: [Junior/Intermédiaire/Senior]

## Prérequis
- [ ] Accès au cluster Kafka
- [ ] Droits d'administration
- [ ] Outils: kafka-cli, kubectl, etc.

## Quand utiliser ce runbook
- Situation 1
- Situation 2

## Procédure

### Étape 1: [Titre]
```bash
# Commandes
```

 **Validation** : [Comment vérifier le succès]
 **En cas d'échec** : [Que faire]

### Étape 2: [Titre]

...

## Rollback

En cas de problème, exécuter les étapes suivantes:

1. ...
2. ...

## Escalade

Si la procédure échoue après 3 tentatives:

* Contacter: [Équipe]
* Slack: #kafka-incidents
* PagerDuty: [Service]

## Historique des modifications

| Date | Auteur | Modification |
| ---- | ------ | ------------ |

```

> **Note de terrain**  
> *Contexte* : Incident majeur dans une banque où le cluster Kafka primaire est devenu indisponible.  
> *Défi* : Les runbooks n'avaient pas été testés depuis 8 mois et contenaient des références à des serveurs décommissionnés.  
> *Solution* : Improvisation par l'équipe senior, basculement réussi en 45 minutes au lieu des 15 minutes prévues.  
> *Leçon* : Les runbooks sont périssables. Intégrer leur validation dans les tests DR réguliers et automatiser leur mise à jour lors des changements d'infrastructure.

---

## III.11.6 Résumé

Ce chapitre a couvert les dimensions essentielles de l'exploitation d'un cluster Kafka en production, depuis les mises à niveau jusqu'à la reprise après sinistre.

### Points clés à retenir

**Évolution et mises à niveau**

La gestion des versions Kafka exige une politique claire qui équilibre stabilité et modernité. La migration vers KRaft représente une évolution majeure qui simplifie l'architecture tout en améliorant la scalabilité. Les rolling upgrades permettent des mises à niveau sans interruption, à condition de respecter la séquence binaires → protocole inter-broker → format des messages.

| Phase | Action | Risque de rollback |
|-------|--------|-------------------|
| 1 | Mise à niveau binaires | Facile |
| 2 | Protocole inter-broker | Modéré |
| 3 | Format messages | Difficile |

**Mobilité des données**

MirrorMaker 2 constitue l'outil de référence pour la réplication inter-clusters. La surveillance du lag de réplication est critique pour les architectures DR. Les migrations de topics requièrent une planification minutieuse incluant la gestion des offsets consommateurs et la validation de l'intégrité des données.

**Surveillance**

L'observabilité Kafka repose sur trois piliers : métriques JMX, logs centralisés et traces distribuées. Les métriques critiques incluent les partitions sous-répliquées, le lag des consommateurs et les performances des brokers. L'automatisation des alertes permet une détection précoce des anomalies.

**Optimisation des performances**

Le diagnostic des problèmes de performance suit une méthodologie systématique. Les leviers principaux d'optimisation sont :
- **Producers** : batching (batch.size, linger.ms), compression (lz4/zstd)
- **Consumers** : parallélisme (partitions), fetch size, traitement asynchrone
- **Brokers** : threads de traitement, configuration OS, gestion des hot partitions

**Reprise après sinistre**

La stratégie DR définit les objectifs RTO et RPO qui guident l'architecture. Les procédures de basculement doivent être documentées dans des runbooks testés régulièrement. Le failback requiert une resynchronisation complète avant de rediriger le trafic vers le cluster primaire.

### Recommandations pratiques

1. **Planifier les mises à niveau** : Établir un calendrier de mise à niveau qui maintient le cluster dans les versions supportées. Ne pas accumuler de retard qui complique les migrations futures.

2. **Automatiser la surveillance** : Déployer une stack d'observabilité complète dès le démarrage du projet. Les métriques historiques sont précieuses pour le diagnostic.

3. **Documenter les procédures** : Maintenir des runbooks à jour pour toutes les opérations critiques. Tester régulièrement leur validité.

4. **Pratiquer le DR** : Les tests de reprise après sinistre révèlent les failles avant les incidents réels. Planifier des tests trimestriels au minimum.

5. **Investir dans l'automatisation** : Les scripts d'opération réduisent les erreurs humaines et accélèrent la résolution des incidents.

### Perspectives

L'exploitation Kafka évolue vers une automatisation croissante. Les opérateurs Kubernetes comme Strimzi et Confluent for Kubernetes simplifient la gestion des clusters. L'observabilité s'enrichit avec l'intégration de l'IA pour la détection d'anomalies et la prédiction des incidents.

Le chapitre suivant explore l'avenir de Kafka : évolutions architecturales, intégration avec l'intelligence artificielle et nouvelles frontières du streaming événementiel. Ces perspectives éclairent les décisions d'investissement à long terme dans la plateforme.

---

*Fin du Chapitre III.11*
```
