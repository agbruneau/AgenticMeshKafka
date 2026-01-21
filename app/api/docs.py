"""
API Documentation - Routes pour la documentation intégrée.

Fournit:
- Recherche full-text dans les patterns et le glossaire
- Liste des patterns par pilier
- Glossaire avec termes et définitions
"""

import json
from pathlib import Path
from typing import List, Optional
from fastapi import APIRouter, Query, HTTPException

router = APIRouter(prefix="/api/docs", tags=["Documentation"])

# Chemins vers les fichiers de documentation
DOCS_PATH = Path(__file__).parent.parent / "docs"
GLOSSARY_PATH = DOCS_PATH / "glossary.json"
PATTERNS_PATH = DOCS_PATH / "patterns.json"


def load_glossary() -> dict:
    """Charge le glossaire depuis le fichier JSON."""
    try:
        if GLOSSARY_PATH.exists():
            return json.loads(GLOSSARY_PATH.read_text(encoding="utf-8"))
        return {"terms": [], "pillars": {}}
    except Exception:
        return {"terms": [], "pillars": {}}


def load_patterns() -> dict:
    """Charge les patterns depuis le fichier JSON."""
    try:
        if PATTERNS_PATH.exists():
            return json.loads(PATTERNS_PATH.read_text(encoding="utf-8"))
        return {"patterns": []}
    except Exception:
        return {"patterns": []}


@router.get("/search")
async def search_docs(
    q: str = Query(..., min_length=2, description="Terme de recherche"),
    pillar: Optional[str] = Query(None, description="Filtrer par pilier")
) -> List[dict]:
    """
    Recherche full-text dans la documentation (patterns + glossaire).

    Args:
        q: Terme de recherche (min 2 caractères)
        pillar: Filtre optionnel par pilier (applications, events, data, cross_cutting)

    Returns:
        Liste des résultats correspondants
    """
    q_lower = q.lower()
    results = []

    # Recherche dans les patterns
    patterns_data = load_patterns()
    for pattern in patterns_data.get("patterns", []):
        if pillar and pattern.get("pillar") != pillar:
            continue

        # Recherche dans nom, problème, solution
        searchable = f"{pattern.get('name', '')} {pattern.get('problem', '')} {pattern.get('solution', '')}".lower()
        if q_lower in searchable:
            results.append({
                "type": "pattern",
                "id": pattern.get("id"),
                "name": pattern.get("name"),
                "pillar": pattern.get("pillar"),
                "category": pattern.get("category"),
                "excerpt": pattern.get("problem", "")[:150] + "..."
            })

    # Recherche dans le glossaire
    glossary_data = load_glossary()
    for term in glossary_data.get("terms", []):
        if pillar and term.get("pillar") != pillar:
            continue

        # Recherche dans terme, définition, aliases
        searchable = f"{term.get('term', '')} {term.get('definition', '')} {' '.join(term.get('aliases', []))}".lower()
        if q_lower in searchable:
            results.append({
                "type": "glossary",
                "id": term.get("id"),
                "name": term.get("term"),
                "pillar": term.get("pillar"),
                "excerpt": term.get("definition", "")[:150] + "..."
            })

    return results


@router.get("/patterns")
async def get_patterns(
    pillar: Optional[str] = Query(None, description="Filtrer par pilier"),
    category: Optional[str] = Query(None, description="Filtrer par catégorie")
) -> List[dict]:
    """
    Liste tous les patterns avec métadonnées.

    Args:
        pillar: Filtre optionnel par pilier
        category: Filtre optionnel par catégorie

    Returns:
        Liste des patterns
    """
    patterns_data = load_patterns()
    patterns = patterns_data.get("patterns", [])

    if pillar:
        patterns = [p for p in patterns if p.get("pillar") == pillar]

    if category:
        patterns = [p for p in patterns if p.get("category", "").lower() == category.lower()]

    return patterns


@router.get("/patterns/{pattern_id}")
async def get_pattern(pattern_id: str) -> dict:
    """
    Retourne les détails d'un pattern spécifique.

    Args:
        pattern_id: Identifiant du pattern

    Returns:
        Détails complets du pattern

    Raises:
        HTTPException: Si le pattern n'existe pas
    """
    patterns_data = load_patterns()

    for pattern in patterns_data.get("patterns", []):
        if pattern.get("id") == pattern_id:
            return pattern

    raise HTTPException(status_code=404, detail=f"Pattern {pattern_id} not found")


@router.get("/glossary")
async def get_glossary(
    pillar: Optional[str] = Query(None, description="Filtrer par pilier")
) -> dict:
    """
    Retourne le glossaire complet.

    Args:
        pillar: Filtre optionnel par pilier

    Returns:
        Glossaire avec termes et pilliers
    """
    glossary_data = load_glossary()

    if pillar:
        glossary_data["terms"] = [
            t for t in glossary_data.get("terms", [])
            if t.get("pillar") == pillar
        ]

    return glossary_data


@router.get("/glossary/{term_id}")
async def get_term(term_id: str) -> dict:
    """
    Retourne la définition d'un terme spécifique.

    Args:
        term_id: Identifiant du terme

    Returns:
        Définition complète du terme

    Raises:
        HTTPException: Si le terme n'existe pas
    """
    glossary_data = load_glossary()

    for term in glossary_data.get("terms", []):
        if term.get("id") == term_id:
            return term

    raise HTTPException(status_code=404, detail=f"Term {term_id} not found")


@router.get("/pillars")
async def get_pillars() -> dict:
    """
    Retourne les métadonnées des piliers (nom, icône, couleur).

    Returns:
        Dictionnaire des piliers avec leurs propriétés
    """
    glossary_data = load_glossary()
    return glossary_data.get("pillars", {})


@router.get("/stats")
async def get_docs_stats() -> dict:
    """
    Retourne des statistiques sur la documentation.

    Returns:
        Nombre de patterns, termes, par pilier
    """
    patterns_data = load_patterns()
    glossary_data = load_glossary()

    patterns = patterns_data.get("patterns", [])
    terms = glossary_data.get("terms", [])

    # Compter par pilier
    patterns_by_pillar = {}
    for p in patterns:
        pillar = p.get("pillar", "other")
        patterns_by_pillar[pillar] = patterns_by_pillar.get(pillar, 0) + 1

    terms_by_pillar = {}
    for t in terms:
        pillar = t.get("pillar", "other")
        terms_by_pillar[pillar] = terms_by_pillar.get(pillar, 0) + 1

    return {
        "total_patterns": len(patterns),
        "total_terms": len(terms),
        "patterns_by_pillar": patterns_by_pillar,
        "terms_by_pillar": terms_by_pillar,
        "categories": list(set(p.get("category", "Other") for p in patterns))
    }


@router.get("/related/{module_id}")
async def get_related_docs(module_id: int) -> dict:
    """
    Retourne la documentation liée à un module théorique.

    Args:
        module_id: ID du module (1-16)

    Returns:
        Patterns et termes liés au module
    """
    # Mapping module -> patterns/termes pertinents
    module_mappings = {
        1: {"patterns": [], "terms": ["interoperability", "coupling"]},
        2: {"patterns": [], "terms": ["pas", "policy", "quote", "claim", "premium"]},
        3: {"patterns": ["api-gateway"], "terms": ["api-gateway", "rest"]},
        4: {"patterns": ["api-gateway", "bff"], "terms": ["api-gateway", "bff", "rate-limiting"]},
        5: {"patterns": ["api-composition", "anti-corruption-layer", "strangler-fig"], "terms": ["api-composition", "acl"]},
        6: {"patterns": ["pubsub", "message-queue"], "terms": ["pubsub", "message-queue", "at-least-once"]},
        7: {"patterns": ["event-sourcing", "cqrs"], "terms": ["event-sourcing", "cqrs"]},
        8: {"patterns": ["saga", "outbox", "dead-letter-queue"], "terms": ["saga", "outbox", "dlq", "compensation"]},
        9: {"patterns": ["etl"], "terms": ["etl"]},
        10: {"patterns": ["cdc"], "terms": ["cdc"]},
        11: {"patterns": ["mdm", "data-quality", "data-lineage"], "terms": ["mdm", "golden-record", "data-lineage", "data-quality"]},
        12: {"patterns": ["circuit-breaker", "retry-backoff"], "terms": ["circuit-breaker", "retry", "fallback", "bulkhead"]},
        13: {"patterns": ["distributed-tracing"], "terms": ["distributed-tracing", "observability"]},
        14: {"patterns": ["jwt-authentication", "rate-limiting"], "terms": ["jwt", "rate-limiting"]},
        15: {"patterns": [], "terms": ["orchestration", "choreography", "adr"]},
        16: {"patterns": [], "terms": []}
    }

    mapping = module_mappings.get(module_id, {"patterns": [], "terms": []})

    # Récupérer les patterns
    patterns_data = load_patterns()
    related_patterns = [
        p for p in patterns_data.get("patterns", [])
        if p.get("id") in mapping["patterns"]
    ]

    # Récupérer les termes
    glossary_data = load_glossary()
    related_terms = [
        t for t in glossary_data.get("terms", [])
        if t.get("id") in mapping["terms"]
    ]

    return {
        "module_id": module_id,
        "patterns": related_patterns,
        "terms": related_terms
    }
