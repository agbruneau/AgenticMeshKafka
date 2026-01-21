/**
 * Gestionnaire de panneaux redimensionnables
 * Permet de redimensionner les panneaux de l'interface avec persistance
 */

class ResizablePanels {
    constructor(options = {}) {
        this.minWidth = options.minWidth || 200;
        this.maxWidth = options.maxWidth || 600;
        this.persistKey = options.persistKey || 'panel-sizes';
        this.panels = new Map();
        this.activeResize = null;

        this.init();
    }

    init() {
        // Charger les tailles sauvegardées
        this.loadSizes();

        // Trouver tous les panneaux redimensionnables
        document.querySelectorAll('[data-resizable]').forEach(panel => {
            this.setupPanel(panel);
        });

        // Listeners globaux pour le drag
        document.addEventListener('mousemove', this.onMouseMove.bind(this));
        document.addEventListener('mouseup', this.onMouseUp.bind(this));

        // Touch support
        document.addEventListener('touchmove', this.onTouchMove.bind(this), { passive: false });
        document.addEventListener('touchend', this.onMouseUp.bind(this));
    }

    setupPanel(panel) {
        const panelId = panel.dataset.resizable;
        const direction = panel.dataset.resizeDirection || 'right';

        // Créer la poignée de redimensionnement
        const handle = document.createElement('div');
        handle.className = `resize-handle resize-handle-${direction}`;
        handle.innerHTML = '<div class="resize-handle-bar"></div>';

        // Positionner la poignée selon la direction
        if (direction === 'right') {
            handle.style.cssText = 'position: absolute; right: 0; top: 0; bottom: 0; width: 6px; cursor: ew-resize; z-index: 10;';
        } else if (direction === 'left') {
            handle.style.cssText = 'position: absolute; left: 0; top: 0; bottom: 0; width: 6px; cursor: ew-resize; z-index: 10;';
        } else if (direction === 'bottom') {
            handle.style.cssText = 'position: absolute; left: 0; right: 0; bottom: 0; height: 6px; cursor: ns-resize; z-index: 10;';
        }

        // Style de la barre de poignée
        const bar = handle.querySelector('.resize-handle-bar');
        bar.style.cssText = direction === 'bottom'
            ? 'width: 50px; height: 4px; margin: auto; background: #4B5563; border-radius: 2px; margin-top: 1px;'
            : 'width: 4px; height: 50px; margin: auto; background: #4B5563; border-radius: 2px; margin-left: 1px;';

        // Hover effect
        handle.addEventListener('mouseenter', () => {
            bar.style.background = '#60A5FA';
        });
        handle.addEventListener('mouseleave', () => {
            if (!this.activeResize) {
                bar.style.background = '#4B5563';
            }
        });

        // Ajouter au panel
        panel.style.position = 'relative';
        panel.appendChild(handle);

        // Event listeners
        handle.addEventListener('mousedown', (e) => this.onMouseDown(e, panelId, panel, direction));
        handle.addEventListener('touchstart', (e) => this.onTouchStart(e, panelId, panel, direction), { passive: false });

        // Appliquer la taille sauvegardée
        if (this.panels.has(panelId)) {
            this.applySize(panel, direction, this.panels.get(panelId));
        }
    }

    onMouseDown(e, panelId, panel, direction) {
        e.preventDefault();
        this.startResize(e.clientX, e.clientY, panelId, panel, direction);
    }

    onTouchStart(e, panelId, panel, direction) {
        e.preventDefault();
        const touch = e.touches[0];
        this.startResize(touch.clientX, touch.clientY, panelId, panel, direction);
    }

    startResize(startX, startY, panelId, panel, direction) {
        const rect = panel.getBoundingClientRect();
        this.activeResize = {
            panelId,
            panel,
            direction,
            startX,
            startY,
            startWidth: rect.width,
            startHeight: rect.height
        };

        document.body.style.cursor = direction === 'bottom' ? 'ns-resize' : 'ew-resize';
        document.body.style.userSelect = 'none';
    }

    onMouseMove(e) {
        if (!this.activeResize) return;
        this.resize(e.clientX, e.clientY);
    }

    onTouchMove(e) {
        if (!this.activeResize) return;
        e.preventDefault();
        const touch = e.touches[0];
        this.resize(touch.clientX, touch.clientY);
    }

    resize(currentX, currentY) {
        const { panel, direction, startX, startY, startWidth, startHeight } = this.activeResize;

        let newSize;
        if (direction === 'right') {
            newSize = startWidth + (currentX - startX);
        } else if (direction === 'left') {
            newSize = startWidth - (currentX - startX);
        } else if (direction === 'bottom') {
            newSize = startHeight + (currentY - startY);
        }

        // Appliquer les limites
        newSize = Math.max(this.minWidth, Math.min(this.maxWidth, newSize));

        this.applySize(panel, direction, newSize);
        this.panels.set(this.activeResize.panelId, newSize);
    }

    applySize(panel, direction, size) {
        if (direction === 'bottom') {
            panel.style.height = `${size}px`;
        } else {
            panel.style.width = `${size}px`;
        }
    }

    onMouseUp() {
        if (!this.activeResize) return;

        // Sauvegarder
        this.saveSizes();

        // Notifier le serveur si disponible
        this.persistToServer(this.activeResize.panelId, this.panels.get(this.activeResize.panelId));

        this.activeResize = null;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
    }

    loadSizes() {
        try {
            const saved = localStorage.getItem(this.persistKey);
            if (saved) {
                const data = JSON.parse(saved);
                Object.entries(data).forEach(([key, value]) => {
                    this.panels.set(key, value);
                });
            }
        } catch (e) {
            console.warn('Could not load panel sizes:', e);
        }
    }

    saveSizes() {
        try {
            const data = {};
            this.panels.forEach((value, key) => {
                data[key] = value;
            });
            localStorage.setItem(this.persistKey, JSON.stringify(data));
        } catch (e) {
            console.warn('Could not save panel sizes:', e);
        }
    }

    async persistToServer(panelId, width) {
        try {
            await fetch(`/api/preferences/panel-size?panel_id=${panelId}&width=${width}`, {
                method: 'PATCH'
            });
        } catch (e) {
            // Silently fail - local storage is sufficient
        }
    }
}

// Gestionnaire de taille de police
class FontSizeManager {
    constructor() {
        this.minSize = 12;
        this.maxSize = 24;
        this.currentSize = 16;
        this.storageKey = 'font-size';

        this.init();
    }

    init() {
        // Charger la taille sauvegardée
        const saved = localStorage.getItem(this.storageKey);
        if (saved) {
            this.currentSize = parseInt(saved, 10);
            this.apply();
        }
    }

    increase() {
        if (this.currentSize < this.maxSize) {
            this.currentSize += 1;
            this.apply();
            this.save();
        }
    }

    decrease() {
        if (this.currentSize > this.minSize) {
            this.currentSize -= 1;
            this.apply();
            this.save();
        }
    }

    reset() {
        this.currentSize = 16;
        this.apply();
        this.save();
    }

    apply() {
        document.documentElement.style.fontSize = `${this.currentSize}px`;

        // Mettre à jour l'indicateur si présent
        const indicator = document.getElementById('font-size-indicator');
        if (indicator) {
            indicator.textContent = `${this.currentSize}px`;
        }
    }

    save() {
        localStorage.setItem(this.storageKey, this.currentSize.toString());

        // Notifier le serveur
        fetch(`/api/preferences/font-size?size=${this.currentSize}`, {
            method: 'PATCH'
        }).catch(() => { });
    }
}

// Auto-initialisation
document.addEventListener('DOMContentLoaded', () => {
    // Panneaux redimensionnables
    window.resizablePanels = new ResizablePanels({
        minWidth: 200,
        maxWidth: 600
    });

    // Taille de police
    window.fontSizeManager = new FontSizeManager();

    // Ajouter les contrôles de taille de police si le conteneur existe
    const fontControls = document.getElementById('font-size-controls');
    if (fontControls) {
        fontControls.innerHTML = `
            <button onclick="fontSizeManager.decrease()" class="px-2 py-1 bg-gray-700 rounded hover:bg-gray-600" title="Réduire la police">A-</button>
            <span id="font-size-indicator" class="px-2">${window.fontSizeManager.currentSize}px</span>
            <button onclick="fontSizeManager.increase()" class="px-2 py-1 bg-gray-700 rounded hover:bg-gray-600" title="Agrandir la police">A+</button>
            <button onclick="fontSizeManager.reset()" class="px-2 py-1 bg-gray-700 rounded hover:bg-gray-600 ml-2" title="Réinitialiser">↺</button>
        `;
    }
});

// Export pour modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { ResizablePanels, FontSizeManager };
}
