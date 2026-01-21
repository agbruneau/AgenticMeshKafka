/**
 * Decision Matrix - Outil interactif de décision d'architecture
 * Aide à choisir le bon type d'intégration selon le contexte
 */

class DecisionMatrix {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.currentAnswers = {};
        this.questions = this.getQuestions();
        this.recommendations = this.getRecommendations();
        this.currentQuestionIndex = 0;

        if (this.container) {
            this.init();
        }
    }

    getQuestions() {
        return [
            {
                id: 'latency',
                question: "Quelle latence est acceptable pour cette intégration ?",
                options: [
                    { value: 'realtime', label: 'Temps réel (< 100ms)', icon: '⚡' },
                    { value: 'seconds', label: 'Quelques secondes (1-10s)', icon: '⏱️' },
                    { value: 'minutes', label: 'Minutes à heures', icon: '⏰' },
                    { value: 'batch', label: 'Batch (nuit/semaine)', icon: '📦' }
                ]
            },
            {
                id: 'consumers',
                question: "Combien de systèmes doivent recevoir l'information ?",
                options: [
                    { value: 'one', label: 'Un seul système', icon: '1️⃣' },
                    { value: 'few', label: '2-3 systèmes', icon: '🔢' },
                    { value: 'many', label: 'Plusieurs (4+)', icon: '📢' },
                    { value: 'unknown', label: 'Nombre inconnu/variable', icon: '❓' }
                ]
            },
            {
                id: 'response',
                question: "L'appelant a-t-il besoin d'une réponse ?",
                options: [
                    { value: 'immediate', label: 'Oui, immédiatement', icon: '↩️' },
                    { value: 'async', label: 'Oui, mais peut attendre', icon: '📬' },
                    { value: 'notification', label: 'Non, juste notifier', icon: '📣' },
                    { value: 'none', label: 'Non, fire-and-forget', icon: '🚀' }
                ]
            },
            {
                id: 'volume',
                question: "Quel volume de données est concerné ?",
                options: [
                    { value: 'single', label: 'Enregistrement unique', icon: '📄' },
                    { value: 'batch_small', label: 'Lot (100-1000)', icon: '📑' },
                    { value: 'batch_large', label: 'Volume massif (10K+)', icon: '📚' },
                    { value: 'stream', label: 'Flux continu', icon: '🌊' }
                ]
            },
            {
                id: 'consistency',
                question: "Quel niveau de cohérence est requis ?",
                options: [
                    { value: 'strong', label: 'Cohérence forte (ACID)', icon: '🔒' },
                    { value: 'eventual', label: 'Cohérence éventuelle OK', icon: '⏳' },
                    { value: 'best_effort', label: 'Best effort suffisant', icon: '👍' }
                ]
            },
            {
                id: 'failure',
                question: "Que se passe-t-il si l'intégration échoue ?",
                options: [
                    { value: 'critical', label: 'Critique - blocage total', icon: '🚨' },
                    { value: 'degraded', label: 'Mode dégradé acceptable', icon: '⚠️' },
                    { value: 'retry', label: 'Retry automatique OK', icon: '🔄' },
                    { value: 'tolerable', label: 'Perte tolérable', icon: '🆗' }
                ]
            }
        ];
    }

    getRecommendations() {
        return {
            'applications': {
                name: 'Intégration Applications',
                icon: '🔗',
                color: '#3B82F6',
                description: 'API REST/gRPC, Gateway, BFF, Composition',
                patterns: ['API Gateway', 'BFF', 'API Composition', 'Anti-Corruption Layer'],
                useCases: [
                    'Interface utilisateur temps réel',
                    'Requête/réponse synchrone',
                    'Agrégation de données live',
                    'Partenaires B2B'
                ]
            },
            'events': {
                name: 'Intégration Événements',
                icon: '⚡',
                color: '#F97316',
                description: 'Pub/Sub, Event Sourcing, Saga, CQRS',
                patterns: ['Pub/Sub', 'Event Sourcing', 'Saga', 'Outbox', 'CQRS'],
                useCases: [
                    'Découplage entre systèmes',
                    'Plusieurs consommateurs',
                    'Workflows longue durée',
                    'Audit trail complet'
                ]
            },
            'data': {
                name: 'Intégration Données',
                icon: '📊',
                color: '#22C55E',
                description: 'ETL, CDC, Data Pipeline, MDM',
                patterns: ['ETL', 'CDC', 'Data Pipeline', 'MDM', 'Data Quality'],
                useCases: [
                    'Volumes massifs',
                    'Analytics/BI',
                    'Synchronisation batch',
                    'Données de référence'
                ]
            },
            'hybrid': {
                name: 'Approche Hybride',
                icon: '🔀',
                color: '#8B5CF6',
                description: 'Combinaison de plusieurs piliers',
                patterns: ['Orchestration + Chorégraphie', 'API + Events', 'Real-time + Batch'],
                useCases: [
                    'Flux complexes multi-étapes',
                    'Besoins mixtes temps réel + batch',
                    'Écosystème mature'
                ]
            }
        };
    }

    init() {
        this.render();
    }

    render() {
        this.container.innerHTML = `
            <div class="decision-matrix bg-gray-800 rounded-lg p-6">
                <div class="header mb-6">
                    <h3 class="text-xl font-bold text-white flex items-center gap-2">
                        🧭 Matrice de Décision d'Architecture
                    </h3>
                    <p class="text-gray-400 mt-2">
                        Répondez aux questions pour obtenir une recommandation d'intégration
                    </p>
                </div>

                <div class="progress-bar mb-6">
                    <div class="flex justify-between text-sm text-gray-400 mb-2">
                        <span>Question ${this.currentQuestionIndex + 1} / ${this.questions.length}</span>
                        <span>${Math.round((this.currentQuestionIndex / this.questions.length) * 100)}%</span>
                    </div>
                    <div class="h-2 bg-gray-700 rounded-full">
                        <div class="h-full bg-blue-500 rounded-full transition-all duration-300"
                             style="width: ${(this.currentQuestionIndex / this.questions.length) * 100}%"></div>
                    </div>
                </div>

                <div id="question-container"></div>

                <div id="result-container" class="hidden"></div>

                <div class="navigation mt-6 flex justify-between">
                    <button id="prev-btn" class="px-4 py-2 bg-gray-700 text-white rounded hover:bg-gray-600 disabled:opacity-50"
                            ${this.currentQuestionIndex === 0 ? 'disabled' : ''}>
                        ← Précédent
                    </button>
                    <button id="reset-btn" class="px-4 py-2 bg-gray-700 text-white rounded hover:bg-gray-600">
                        🔄 Recommencer
                    </button>
                </div>
            </div>
        `;

        this.renderQuestion();
        this.attachEventListeners();
    }

    renderQuestion() {
        const questionContainer = document.getElementById('question-container');
        const question = this.questions[this.currentQuestionIndex];

        if (!question) {
            this.showResult();
            return;
        }

        const selectedValue = this.currentAnswers[question.id];

        questionContainer.innerHTML = `
            <div class="question animate-fadeIn">
                <h4 class="text-lg font-semibold text-white mb-4">${question.question}</h4>
                <div class="options grid gap-3">
                    ${question.options.map(option => `
                        <button class="option-btn p-4 rounded-lg border-2 transition-all duration-200 text-left
                                       ${selectedValue === option.value
                                         ? 'border-blue-500 bg-blue-900/30'
                                         : 'border-gray-600 hover:border-gray-500 bg-gray-700/50'}"
                                data-value="${option.value}">
                            <span class="text-2xl mr-3">${option.icon}</span>
                            <span class="text-white">${option.label}</span>
                        </button>
                    `).join('')}
                </div>
            </div>
        `;

        // Attach option click handlers
        questionContainer.querySelectorAll('.option-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const value = e.currentTarget.dataset.value;
                this.selectOption(question.id, value);
            });
        });
    }

    selectOption(questionId, value) {
        this.currentAnswers[questionId] = value;

        // Auto-advance to next question
        setTimeout(() => {
            this.currentQuestionIndex++;
            if (this.currentQuestionIndex < this.questions.length) {
                this.render();
            } else {
                this.showResult();
            }
        }, 300);
    }

    calculateRecommendation() {
        const scores = {
            applications: 0,
            events: 0,
            data: 0
        };

        // Latency scoring
        if (this.currentAnswers.latency === 'realtime') {
            scores.applications += 3;
        } else if (this.currentAnswers.latency === 'seconds') {
            scores.applications += 2;
            scores.events += 1;
        } else if (this.currentAnswers.latency === 'minutes') {
            scores.events += 2;
            scores.data += 1;
        } else if (this.currentAnswers.latency === 'batch') {
            scores.data += 3;
        }

        // Consumers scoring
        if (this.currentAnswers.consumers === 'one') {
            scores.applications += 2;
        } else if (this.currentAnswers.consumers === 'few') {
            scores.events += 1;
            scores.applications += 1;
        } else if (this.currentAnswers.consumers === 'many' || this.currentAnswers.consumers === 'unknown') {
            scores.events += 3;
        }

        // Response scoring
        if (this.currentAnswers.response === 'immediate') {
            scores.applications += 3;
        } else if (this.currentAnswers.response === 'async') {
            scores.events += 2;
            scores.applications += 1;
        } else if (this.currentAnswers.response === 'notification') {
            scores.events += 3;
        } else if (this.currentAnswers.response === 'none') {
            scores.events += 2;
            scores.data += 1;
        }

        // Volume scoring
        if (this.currentAnswers.volume === 'single') {
            scores.applications += 2;
        } else if (this.currentAnswers.volume === 'batch_small') {
            scores.events += 1;
            scores.data += 1;
        } else if (this.currentAnswers.volume === 'batch_large') {
            scores.data += 3;
        } else if (this.currentAnswers.volume === 'stream') {
            scores.events += 2;
            scores.data += 1;
        }

        // Consistency scoring
        if (this.currentAnswers.consistency === 'strong') {
            scores.applications += 2;
        } else if (this.currentAnswers.consistency === 'eventual') {
            scores.events += 2;
            scores.data += 1;
        } else if (this.currentAnswers.consistency === 'best_effort') {
            scores.events += 1;
            scores.data += 1;
        }

        // Failure scoring
        if (this.currentAnswers.failure === 'critical') {
            scores.applications += 2;
        } else if (this.currentAnswers.failure === 'degraded') {
            scores.applications += 1;
            scores.events += 1;
        } else if (this.currentAnswers.failure === 'retry') {
            scores.events += 2;
        } else if (this.currentAnswers.failure === 'tolerable') {
            scores.data += 1;
            scores.events += 1;
        }

        // Determine winner
        const maxScore = Math.max(scores.applications, scores.events, scores.data);
        const totalScore = scores.applications + scores.events + scores.data;

        // Check for hybrid recommendation (close scores)
        const threshold = maxScore * 0.7;
        const closeScores = Object.values(scores).filter(s => s >= threshold).length;

        if (closeScores >= 2) {
            return {
                primary: 'hybrid',
                scores: scores,
                confidence: 'medium'
            };
        }

        let primary = 'applications';
        if (scores.events === maxScore) primary = 'events';
        if (scores.data === maxScore) primary = 'data';

        const confidence = maxScore / totalScore > 0.5 ? 'high' : 'medium';

        return {
            primary: primary,
            scores: scores,
            confidence: confidence
        };
    }

    showResult() {
        const questionContainer = document.getElementById('question-container');
        const resultContainer = document.getElementById('result-container');

        questionContainer.classList.add('hidden');
        resultContainer.classList.remove('hidden');

        const result = this.calculateRecommendation();
        const recommendation = this.recommendations[result.primary];

        resultContainer.innerHTML = `
            <div class="result animate-fadeIn">
                <div class="recommendation-card p-6 rounded-lg border-2 mb-6"
                     style="border-color: ${recommendation.color}; background: ${recommendation.color}20">
                    <div class="flex items-center gap-4 mb-4">
                        <span class="text-5xl">${recommendation.icon}</span>
                        <div>
                            <h3 class="text-2xl font-bold text-white">${recommendation.name}</h3>
                            <p class="text-gray-300">${recommendation.description}</p>
                        </div>
                    </div>

                    <div class="confidence mb-4">
                        <span class="text-sm text-gray-400">Confiance: </span>
                        <span class="px-2 py-1 rounded text-sm ${result.confidence === 'high' ? 'bg-green-600' : 'bg-yellow-600'} text-white">
                            ${result.confidence === 'high' ? 'Élevée' : 'Moyenne'}
                        </span>
                    </div>

                    <div class="patterns mb-4">
                        <h4 class="text-white font-semibold mb-2">Patterns recommandés :</h4>
                        <div class="flex flex-wrap gap-2">
                            ${recommendation.patterns.map(p => `
                                <span class="px-3 py-1 bg-gray-700 text-gray-200 rounded-full text-sm">${p}</span>
                            `).join('')}
                        </div>
                    </div>

                    <div class="use-cases">
                        <h4 class="text-white font-semibold mb-2">Cas d'usage typiques :</h4>
                        <ul class="text-gray-300 space-y-1">
                            ${recommendation.useCases.map(uc => `
                                <li class="flex items-center gap-2">
                                    <span class="text-green-400">✓</span> ${uc}
                                </li>
                            `).join('')}
                        </ul>
                    </div>
                </div>

                <div class="scores-chart">
                    <h4 class="text-white font-semibold mb-3">Analyse des scores :</h4>
                    <div class="space-y-3">
                        ${this.renderScoreBar('Applications', result.scores.applications, '#3B82F6')}
                        ${this.renderScoreBar('Événements', result.scores.events, '#F97316')}
                        ${this.renderScoreBar('Données', result.scores.data, '#22C55E')}
                    </div>
                </div>

                <div class="summary mt-6 p-4 bg-gray-700/50 rounded-lg">
                    <h4 class="text-white font-semibold mb-2">📋 Résumé de vos choix :</h4>
                    <ul class="text-gray-300 text-sm space-y-1">
                        ${this.renderAnswersSummary()}
                    </ul>
                </div>
            </div>
        `;

        // Update progress bar to 100%
        const progressBar = this.container.querySelector('.progress-bar .bg-blue-500');
        if (progressBar) {
            progressBar.style.width = '100%';
        }
    }

    renderScoreBar(label, score, color) {
        const maxPossibleScore = 15; // Approximate max
        const percentage = Math.min((score / maxPossibleScore) * 100, 100);

        return `
            <div class="score-bar">
                <div class="flex justify-between text-sm mb-1">
                    <span class="text-gray-300">${label}</span>
                    <span class="text-gray-400">${score} pts</span>
                </div>
                <div class="h-3 bg-gray-700 rounded-full overflow-hidden">
                    <div class="h-full rounded-full transition-all duration-500"
                         style="width: ${percentage}%; background-color: ${color}"></div>
                </div>
            </div>
        `;
    }

    renderAnswersSummary() {
        const labels = {
            latency: { label: 'Latence', realtime: 'Temps réel', seconds: 'Secondes', minutes: 'Minutes', batch: 'Batch' },
            consumers: { label: 'Consommateurs', one: 'Un seul', few: '2-3', many: 'Plusieurs', unknown: 'Variable' },
            response: { label: 'Réponse', immediate: 'Immédiate', async: 'Asynchrone', notification: 'Notification', none: 'Aucune' },
            volume: { label: 'Volume', single: 'Unique', batch_small: 'Petit lot', batch_large: 'Massif', stream: 'Flux' },
            consistency: { label: 'Cohérence', strong: 'Forte', eventual: 'Éventuelle', best_effort: 'Best effort' },
            failure: { label: 'Échec', critical: 'Critique', degraded: 'Dégradé OK', retry: 'Retry OK', tolerable: 'Tolérable' }
        };

        return Object.entries(this.currentAnswers).map(([key, value]) => {
            const config = labels[key];
            return `<li><strong>${config.label}:</strong> ${config[value] || value}</li>`;
        }).join('');
    }

    attachEventListeners() {
        const prevBtn = document.getElementById('prev-btn');
        const resetBtn = document.getElementById('reset-btn');

        if (prevBtn) {
            prevBtn.addEventListener('click', () => {
                if (this.currentQuestionIndex > 0) {
                    this.currentQuestionIndex--;
                    this.render();
                }
            });
        }

        if (resetBtn) {
            resetBtn.addEventListener('click', () => {
                this.currentAnswers = {};
                this.currentQuestionIndex = 0;
                this.render();
            });
        }
    }
}

// Auto-initialize if container exists
document.addEventListener('DOMContentLoaded', () => {
    const container = document.getElementById('decision-matrix');
    if (container) {
        window.decisionMatrix = new DecisionMatrix('decision-matrix');
    }
});

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = DecisionMatrix;
}
