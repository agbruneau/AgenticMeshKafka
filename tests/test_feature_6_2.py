"""
Tests Feature 6.2 : Documentation Intégrée

Vérifie:
- API /api/docs/search fonctionnelle
- 25+ patterns documentés
- 50+ termes de glossaire
- Navigation et recherche
"""

import pytest
from pathlib import Path
import json


# Tests des fichiers de documentation

def test_glossary_file_exists():
    """Vérifie que le fichier glossary.json existe."""
    filepath = Path("app/docs/glossary.json")
    assert filepath.exists(), "glossary.json should exist"


def test_glossary_has_50_terms():
    """Vérifie que le glossaire contient 50+ termes."""
    filepath = Path("app/docs/glossary.json")
    content = json.loads(filepath.read_text(encoding="utf-8"))
    terms = content.get("terms", [])
    assert len(terms) >= 50, f"Glossary should have at least 50 terms, found {len(terms)}"


def test_glossary_term_structure():
    """Vérifie la structure des termes du glossaire."""
    filepath = Path("app/docs/glossary.json")
    content = json.loads(filepath.read_text(encoding="utf-8"))
    terms = content.get("terms", [])

    required_fields = ["id", "term", "definition", "pillar"]
    for term in terms[:10]:  # Vérifier les 10 premiers
        for field in required_fields:
            assert field in term, f"Term should have '{field}' field"


def test_glossary_covers_all_pillars():
    """Vérifie que le glossaire couvre les 3 piliers."""
    filepath = Path("app/docs/glossary.json")
    content = json.loads(filepath.read_text(encoding="utf-8"))
    terms = content.get("terms", [])

    pillars = set(term.get("pillar") for term in terms)
    assert "applications" in pillars, "Glossary should cover Applications pillar"
    assert "events" in pillars, "Glossary should cover Events pillar"
    assert "data" in pillars, "Glossary should cover Data pillar"


def test_patterns_file_exists():
    """Vérifie que le fichier patterns.json existe."""
    filepath = Path("app/docs/patterns.json")
    assert filepath.exists(), "patterns.json should exist"


def test_patterns_has_25_patterns():
    """Vérifie qu'il y a 25+ patterns documentés."""
    filepath = Path("app/docs/patterns.json")
    content = json.loads(filepath.read_text(encoding="utf-8"))
    patterns = content.get("patterns", [])
    assert len(patterns) >= 25, f"Should have at least 25 patterns, found {len(patterns)}"


def test_patterns_structure():
    """Vérifie la structure des patterns."""
    filepath = Path("app/docs/patterns.json")
    content = json.loads(filepath.read_text(encoding="utf-8"))
    patterns = content.get("patterns", [])

    required_fields = ["id", "name", "pillar", "problem", "solution"]
    for pattern in patterns[:10]:  # Vérifier les 10 premiers
        for field in required_fields:
            assert field in pattern, f"Pattern should have '{field}' field"


def test_patterns_have_insurance_examples():
    """Vérifie que les patterns ont des exemples assurance."""
    filepath = Path("app/docs/patterns.json")
    content = json.loads(filepath.read_text(encoding="utf-8"))
    patterns = content.get("patterns", [])

    patterns_with_examples = sum(1 for p in patterns if p.get("insurance_example"))
    assert patterns_with_examples >= 20, f"At least 20 patterns should have insurance examples, found {patterns_with_examples}"


def test_docs_api_router_exists():
    """Vérifie que le router docs existe."""
    filepath = Path("app/api/docs.py")
    assert filepath.exists(), "docs.py router should exist"
    content = filepath.read_text(encoding="utf-8")
    assert "router" in content
    assert "/search" in content
    assert "/patterns" in content
    assert "/glossary" in content


# Tests API

@pytest.mark.asyncio
async def test_docs_search_api(client):
    """Vérifie que l'API de recherche fonctionne."""
    async with client:
        r = await client.get("/api/docs/search?q=api")
        assert r.status_code == 200
        data = r.json()
        # API returns list directly or dict with results key
        results = data.get("results", data) if isinstance(data, dict) else data
        assert len(results) > 0, "Search for 'api' should return results"


@pytest.mark.asyncio
async def test_docs_search_no_query(client):
    """Vérifie la recherche sans terme retourne une erreur ou résultat vide."""
    async with client:
        r = await client.get("/api/docs/search?q=")
        # Accept 400, 422, or 200 with empty results
        assert r.status_code in [200, 400, 422]


@pytest.mark.asyncio
async def test_docs_patterns_list(client):
    """Vérifie que l'API liste les patterns."""
    async with client:
        r = await client.get("/api/docs/patterns")
        assert r.status_code == 200
        data = r.json()
        # API may return list directly or dict with patterns key
        patterns = data.get("patterns", data) if isinstance(data, dict) else data
        assert len(patterns) >= 25, f"API should return 25+ patterns, found {len(patterns)}"


@pytest.mark.asyncio
async def test_docs_patterns_filter_by_pillar(client):
    """Vérifie le filtrage des patterns par pilier."""
    async with client:
        r = await client.get("/api/docs/patterns?pillar=applications")
        assert r.status_code == 200
        data = r.json()
        # API may return list directly or dict with patterns key
        patterns = data.get("patterns", data) if isinstance(data, dict) else data
        # All returned patterns should be for applications pillar
        for pattern in patterns:
            assert pattern.get("pillar") == "applications"


@pytest.mark.asyncio
async def test_docs_pattern_detail(client):
    """Vérifie l'accès au détail d'un pattern."""
    async with client:
        r = await client.get("/api/docs/patterns/api-gateway")
        assert r.status_code == 200
        data = r.json()
        assert "name" in data
        assert "problem" in data
        assert "solution" in data


@pytest.mark.asyncio
async def test_docs_glossary_api(client):
    """Vérifie que l'API glossaire fonctionne."""
    async with client:
        r = await client.get("/api/docs/glossary")
        assert r.status_code == 200
        data = r.json()
        assert "terms" in data
        assert len(data["terms"]) >= 50, f"API should return 50+ terms, found {len(data['terms'])}"


@pytest.mark.asyncio
async def test_docs_glossary_filter_by_pillar(client):
    """Vérifie le filtrage du glossaire par pilier."""
    async with client:
        r = await client.get("/api/docs/glossary?pillar=events")
        assert r.status_code == 200
        data = r.json()
        assert "terms" in data
        for term in data["terms"]:
            assert term.get("pillar") == "events"


@pytest.mark.asyncio
async def test_docs_term_detail(client):
    """Vérifie l'accès au détail d'un terme."""
    async with client:
        r = await client.get("/api/docs/glossary/api-gateway")
        assert r.status_code == 200
        data = r.json()
        assert "term" in data
        assert "definition" in data


@pytest.mark.asyncio
async def test_docs_pillars_api(client):
    """Vérifie l'API des piliers."""
    async with client:
        r = await client.get("/api/docs/pillars")
        assert r.status_code == 200
        data = r.json()
        # API may return dict with pillars key or dict with pillar ids as keys
        if "pillars" in data:
            pillar_ids = [p.get("id") for p in data["pillars"]]
        else:
            # Dict format with pillar ids as keys
            pillar_ids = list(data.keys())
        assert "applications" in pillar_ids
        assert "events" in pillar_ids
        assert "data" in pillar_ids


@pytest.mark.asyncio
async def test_docs_stats_api(client):
    """Vérifie l'API de statistiques."""
    async with client:
        r = await client.get("/api/docs/stats")
        assert r.status_code == 200
        data = r.json()
        assert "total_patterns" in data
        assert "total_terms" in data
        assert data["total_patterns"] >= 25
        assert data["total_terms"] >= 50


@pytest.mark.asyncio
async def test_docs_related_for_module(client):
    """Vérifie les documents liés à un module."""
    async with client:
        r = await client.get("/api/docs/related/1")  # Module 1
        assert r.status_code == 200
        data = r.json()
        assert "patterns" in data or "terms" in data
