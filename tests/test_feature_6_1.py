"""
Tests Feature 6.1 : Modules 15-16 & Projet Final

Vérifie:
- Module 15 (Architecture Decisions) avec 5 sections
- Module 16 (Projet Final) avec 5 sections
- Scénario CROSS-04 avec 10+ étapes
"""

import pytest
from pathlib import Path


# Tests des fichiers de contenu

def test_module15_directory_exists():
    """Vérifie que le dossier du module 15 existe."""
    module_dir = Path("app/theory/content/15_architecture_decisions")
    assert module_dir.is_dir(), "Module 15 directory should exist"


def test_module15_has_5_sections():
    """Vérifie que le module 15 a 5 sections."""
    module_dir = Path("app/theory/content/15_architecture_decisions")
    md_files = list(module_dir.glob("*.md"))
    assert len(md_files) >= 5, f"Module 15 should have at least 5 sections, found {len(md_files)}"


def test_module15_content_files():
    """Vérifie les fichiers de contenu du module 15."""
    base = Path("app/theory/content/15_architecture_decisions")
    expected_files = [
        "01_orchestration_vs_choreography.md",
        "02_quand_utiliser.md",
        "03_tradeoffs.md",
        "04_adr.md",
        "05_antipatterns.md"
    ]
    for filename in expected_files:
        filepath = base / filename
        assert filepath.exists(), f"Missing file: {filename}"
        content = filepath.read_text(encoding="utf-8")
        assert len(content) > 500, f"File {filename} should have substantial content"


def test_module15_orchestration_content():
    """Vérifie le contenu sur orchestration vs chorégraphie."""
    filepath = Path("app/theory/content/15_architecture_decisions/01_orchestration_vs_choreography.md")
    content = filepath.read_text(encoding="utf-8").lower()
    assert "orchestration" in content
    assert "choreograph" in content or "chorégraph" in content
    assert "saga" in content


def test_module15_antipatterns_content():
    """Vérifie le contenu sur les anti-patterns."""
    filepath = Path("app/theory/content/15_architecture_decisions/05_antipatterns.md")
    content = filepath.read_text(encoding="utf-8").lower()
    assert "spaghetti" in content
    assert "god service" in content or "god" in content
    assert "distributed monolith" in content or "monolith" in content


def test_module16_directory_exists():
    """Vérifie que le dossier du module 16 existe."""
    module_dir = Path("app/theory/content/16_projet_final")
    assert module_dir.is_dir(), "Module 16 directory should exist"


def test_module16_has_5_sections():
    """Vérifie que le module 16 a 5 sections."""
    module_dir = Path("app/theory/content/16_projet_final")
    md_files = list(module_dir.glob("*.md"))
    assert len(md_files) >= 5, f"Module 16 should have at least 5 sections, found {len(md_files)}"


def test_module16_content_files():
    """Vérifie les fichiers de contenu du module 16."""
    base = Path("app/theory/content/16_projet_final")
    expected_files = [
        "01_cahier_des_charges.md",
        "02_conception_architecture.md",
        "03_implementation.md",
        "04_tests_validation.md",
        "05_evaluation_finale.md"
    ]
    for filename in expected_files:
        filepath = base / filename
        assert filepath.exists(), f"Missing file: {filename}"
        content = filepath.read_text(encoding="utf-8")
        assert len(content) > 500, f"File {filename} should have substantial content"


def test_decision_matrix_exists():
    """Vérifie que le fichier de la matrice de décision existe."""
    filepath = Path("static/js/decision-matrix.js")
    assert filepath.exists(), "Decision matrix JS should exist"
    content = filepath.read_text(encoding="utf-8")
    assert "DecisionMatrix" in content
    assert "getQuestions" in content
    assert "calculateRecommendation" in content


def test_cross04_scenario_exists():
    """Vérifie que le scénario CROSS-04 existe."""
    filepath = Path("app/sandbox/scenarios/cross_04.py")
    assert filepath.exists(), "CROSS-04 scenario file should exist"


def test_cross04_scenario_content():
    """Vérifie le contenu du scénario CROSS-04."""
    filepath = Path("app/sandbox/scenarios/cross_04.py")
    content = filepath.read_text(encoding="utf-8")
    assert "Cross04Scenario" in content
    assert "step_1" in content
    assert "step_10" in content
    assert "applications" in content.lower()
    assert "events" in content.lower()
    assert "data" in content.lower()


# Tests API

@pytest.mark.asyncio
async def test_modules_15_16_api(client):
    """Vérifie que les modules 15 et 16 sont accessibles via l'API."""
    async with client:
        for m in [15, 16]:
            r = await client.get(f"/api/theory/modules/{m}")
            assert r.status_code == 200, f"Module {m} should be accessible"
            data = r.json()
            assert "content" in data, f"Module {m} should have content"


@pytest.mark.asyncio
async def test_cross04_scenario_api(client):
    """Vérifie que le scénario CROSS-04 est accessible via l'API."""
    async with client:
        r = await client.get("/api/sandbox/scenarios/CROSS-04")
        assert r.status_code == 200, "CROSS-04 scenario should be accessible"
        data = r.json()
        assert "steps" in data, "Scenario should have steps"
        assert len(data["steps"]) >= 10, f"CROSS-04 should have at least 10 steps, found {len(data['steps'])}"


@pytest.mark.asyncio
async def test_cross04_integrates_three_pillars(client):
    """Vérifie que CROSS-04 intègre les 3 piliers."""
    async with client:
        r = await client.get("/api/sandbox/scenarios/CROSS-04")
        data = r.json()
        steps_text = " ".join([s.get("instruction", "") + s.get("title", "") for s in data.get("steps", [])])
        steps_text = steps_text.lower()

        # Vérifier la présence des concepts des 3 piliers
        has_api = "api" in steps_text or "gateway" in steps_text
        has_events = "event" in steps_text or "publish" in steps_text
        has_data = "data" in steps_text or "sync" in steps_text or "dwh" in steps_text

        assert has_api, "CROSS-04 should include API/Applications concepts"
        assert has_events, "CROSS-04 should include Events concepts"
        assert has_data, "CROSS-04 should include Data concepts"
