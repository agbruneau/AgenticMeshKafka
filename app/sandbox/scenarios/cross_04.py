"""
Scénario CROSS-04: Écosystème Complet
=====================================

Ce scénario intègre les trois piliers d'intégration dans un flux complet
de souscription d'assurance, démontrant comment Applications, Événements
et Données travaillent ensemble.

Flux:
1. 🔗 API Gateway → Quote Engine (calcul devis)
2. 🔗 API Gateway → Policy Admin (création police)
3. ⚡ Event Bus → PolicyCreated (publication)
4. ⚡ Consumers → Billing, Notifications, Documents, Audit
5. 📊 CDC → Data Warehouse (synchronisation)
6. 🛡️ Circuit Breaker → Gestion panne Billing
"""

import asyncio
from datetime import datetime
from typing import Any, Dict, List, Optional
import json


class Cross04Scenario:
    """Scénario d'écosystème complet intégrant les 3 piliers."""

    def __init__(self):
        self.state = {
            "current_step": 0,
            "quote": None,
            "policy": None,
            "events_published": [],
            "events_consumed": [],
            "dwh_synced": False,
            "circuit_breaker_state": "CLOSED",
            "failures": [],
            "recovery_performed": False
        }
        self.trace_id = None
        self.logs = []

    def log(self, level: str, message: str, data: Dict = None):
        """Ajoute un log avec trace_id."""
        entry = {
            "timestamp": datetime.now().isoformat(),
            "level": level,
            "trace_id": self.trace_id,
            "message": message,
            "data": data or {}
        }
        self.logs.append(entry)
        return entry

    async def step_1_receive_quote_request(self, request_data: Dict) -> Dict:
        """
        Étape 1: Réception demande de devis via API Gateway
        Pilier: 🔗 Applications
        """
        import uuid
        self.trace_id = str(uuid.uuid4())[:8]

        self.log("INFO", "Gateway received quote request", {
            "customer_id": request_data.get("customer_id"),
            "product": request_data.get("product")
        })

        # Simulation authentification JWT
        auth_result = {
            "authenticated": True,
            "client_id": request_data.get("client_id", "web-portal"),
            "rate_limit_remaining": 99
        }

        self.log("INFO", "Authentication successful", auth_result)

        self.state["current_step"] = 1
        return {
            "step": 1,
            "pillar": "applications",
            "action": "Gateway authentication",
            "result": "Request authenticated and routed to Quote Engine",
            "trace_id": self.trace_id
        }

    async def step_2_calculate_quote(self, risk_data: Dict) -> Dict:
        """
        Étape 2: Calcul du devis par Quote Engine
        Pilier: 🔗 Applications
        """
        # Simulation appel Rating API (externe)
        rating_api_result = {
            "base_rate": 500,
            "risk_factor": 1.2,
            "discounts": 0.9
        }

        # Simulation calcul de prime
        premium = rating_api_result["base_rate"] * rating_api_result["risk_factor"] * rating_api_result["discounts"]

        self.state["quote"] = {
            "id": f"Q-{datetime.now().strftime('%Y%m%d%H%M%S')}",
            "customer_id": risk_data.get("customer_id"),
            "product": risk_data.get("product", "AUTO"),
            "premium": round(premium, 2),
            "validity": "30 days",
            "status": "VALID"
        }

        self.log("INFO", "Quote calculated", {
            "quote_id": self.state["quote"]["id"],
            "premium": self.state["quote"]["premium"]
        })

        self.state["current_step"] = 2
        return {
            "step": 2,
            "pillar": "applications",
            "action": "Quote calculation",
            "result": f"Quote {self.state['quote']['id']} created with premium {self.state['quote']['premium']}€",
            "data": self.state["quote"]
        }

    async def step_3_create_policy(self) -> Dict:
        """
        Étape 3: Création de la police par Policy Admin
        Pilier: 🔗 Applications (Saga)
        """
        if not self.state["quote"]:
            raise ValueError("No quote available to convert to policy")

        # Simulation saga de souscription
        saga_steps = [
            {"name": "ReserveQuote", "status": "COMPLETED"},
            {"name": "VerifyCustomer", "status": "COMPLETED"},
            {"name": "CreatePolicy", "status": "COMPLETED"},
            {"name": "InitializeBilling", "status": "PENDING"}
        ]

        self.state["policy"] = {
            "number": f"POL-{datetime.now().strftime('%Y%m%d%H%M%S')}",
            "quote_id": self.state["quote"]["id"],
            "customer_id": self.state["quote"]["customer_id"],
            "product": self.state["quote"]["product"],
            "premium": self.state["quote"]["premium"],
            "status": "ACTIVE",
            "start_date": datetime.now().isoformat(),
            "coverages": ["RC", "VOL", "BRIS_GLACE"]
        }

        self.log("INFO", "Policy created via saga", {
            "policy_number": self.state["policy"]["number"],
            "saga_steps": saga_steps
        })

        self.state["current_step"] = 3
        return {
            "step": 3,
            "pillar": "applications",
            "action": "Policy creation (Saga)",
            "result": f"Policy {self.state['policy']['number']} created",
            "saga_steps": saga_steps,
            "data": self.state["policy"]
        }

    async def step_4_publish_policy_created(self) -> Dict:
        """
        Étape 4: Publication de l'événement PolicyCreated
        Pilier: ⚡ Événements (Pub/Sub)
        """
        if not self.state["policy"]:
            raise ValueError("No policy available to publish event for")

        event = {
            "type": "PolicyCreated",
            "timestamp": datetime.now().isoformat(),
            "trace_id": self.trace_id,
            "payload": {
                "policy_number": self.state["policy"]["number"],
                "customer_id": self.state["policy"]["customer_id"],
                "product": self.state["policy"]["product"],
                "premium": self.state["policy"]["premium"],
                "coverages": self.state["policy"]["coverages"]
            }
        }

        self.state["events_published"].append(event)

        self.log("INFO", "Event published to topic.policies", {
            "event_type": event["type"],
            "subscribers": ["billing", "notifications", "documents", "audit"]
        })

        self.state["current_step"] = 4
        return {
            "step": 4,
            "pillar": "events",
            "action": "Pub/Sub - PolicyCreated",
            "result": "Event published to 4 subscribers",
            "event": event,
            "subscribers": ["billing", "notifications", "documents", "audit"]
        }

    async def step_5_billing_consumes(self) -> Dict:
        """
        Étape 5: Billing consomme l'événement et génère une facture
        Pilier: ⚡ Événements (Consumer)
        """
        invoice = {
            "id": f"INV-{datetime.now().strftime('%Y%m%d%H%M%S')}",
            "policy_number": self.state["policy"]["number"],
            "amount": self.state["policy"]["premium"],
            "due_date": "2024-02-15",
            "status": "PENDING"
        }

        consumed_event = {
            "consumer": "billing",
            "event_type": "PolicyCreated",
            "action": "Generate invoice",
            "result": invoice
        }
        self.state["events_consumed"].append(consumed_event)

        self.log("INFO", "Billing consumed PolicyCreated", {
            "invoice_id": invoice["id"],
            "amount": invoice["amount"]
        })

        self.state["current_step"] = 5
        return {
            "step": 5,
            "pillar": "events",
            "action": "Billing consumer",
            "result": f"Invoice {invoice['id']} generated for {invoice['amount']}€",
            "data": invoice
        }

    async def step_6_notifications_consumes(self) -> Dict:
        """
        Étape 6: Notifications envoie un email de bienvenue
        Pilier: ⚡ Événements (Consumer)
        """
        notification = {
            "type": "EMAIL",
            "template": "WELCOME_POLICY",
            "recipient": f"customer_{self.state['policy']['customer_id']}@email.com",
            "subject": f"Votre police {self.state['policy']['number']} est active",
            "status": "SENT"
        }

        consumed_event = {
            "consumer": "notifications",
            "event_type": "PolicyCreated",
            "action": "Send welcome email",
            "result": notification
        }
        self.state["events_consumed"].append(consumed_event)

        self.log("INFO", "Notifications consumed PolicyCreated", {
            "notification_type": notification["type"],
            "recipient": notification["recipient"]
        })

        self.state["current_step"] = 6
        return {
            "step": 6,
            "pillar": "events",
            "action": "Notifications consumer",
            "result": f"Welcome email sent to {notification['recipient']}",
            "data": notification
        }

    async def step_7_cdc_sync(self) -> Dict:
        """
        Étape 7: CDC synchronise les données vers le Data Warehouse
        Pilier: 📊 Données (CDC)
        """
        cdc_change = {
            "table": "policies",
            "operation": "INSERT",
            "timestamp": datetime.now().isoformat(),
            "before": None,
            "after": self.state["policy"]
        }

        # Simulation transformation ETL
        dwh_record = {
            "policy_id": self.state["policy"]["number"],
            "customer_id": self.state["policy"]["customer_id"],
            "product_line": self.state["policy"]["product"],
            "gross_premium": self.state["policy"]["premium"],
            "net_premium": self.state["policy"]["premium"] * 0.85,  # Simule commission
            "coverage_count": len(self.state["policy"]["coverages"]),
            "created_at": self.state["policy"]["start_date"],
            "loaded_at": datetime.now().isoformat()
        }

        self.state["dwh_synced"] = True

        self.log("INFO", "CDC captured and synced to DWH", {
            "operation": cdc_change["operation"],
            "latency_ms": 450
        })

        self.state["current_step"] = 7
        return {
            "step": 7,
            "pillar": "data",
            "action": "CDC → Data Warehouse",
            "result": "Policy synced to DWH with enrichment",
            "cdc_change": cdc_change,
            "dwh_record": dwh_record
        }

    async def step_8_reporting_updated(self) -> Dict:
        """
        Étape 8: Dashboard de reporting mis à jour
        Pilier: 📊 Données (Analytics)
        """
        reporting_update = {
            "dashboard": "daily_sales",
            "metrics_updated": [
                {"name": "policies_created_today", "delta": 1},
                {"name": "premium_volume_today", "delta": self.state["policy"]["premium"]},
                {"name": "auto_policies_count", "delta": 1 if self.state["policy"]["product"] == "AUTO" else 0}
            ],
            "last_refresh": datetime.now().isoformat()
        }

        self.log("INFO", "Reporting dashboard updated", {
            "dashboard": reporting_update["dashboard"],
            "metrics_count": len(reporting_update["metrics_updated"])
        })

        self.state["current_step"] = 8
        return {
            "step": 8,
            "pillar": "data",
            "action": "Reporting refresh",
            "result": "Dashboard updated with new policy metrics",
            "data": reporting_update
        }

    async def step_9_simulate_failure(self) -> Dict:
        """
        Étape 9: Simulation d'une panne du service Billing
        Pilier: 🛡️ Résilience (Circuit Breaker)
        """
        # Simulation de pannes successives
        failures = [
            {"attempt": 1, "error": "Connection timeout", "timestamp": datetime.now().isoformat()},
            {"attempt": 2, "error": "Connection refused", "timestamp": datetime.now().isoformat()},
            {"attempt": 3, "error": "Connection refused", "timestamp": datetime.now().isoformat()},
        ]

        self.state["failures"] = failures
        self.state["circuit_breaker_state"] = "OPEN"

        self.log("WARN", "Circuit breaker OPENED for billing service", {
            "failures_count": len(failures),
            "circuit_state": "OPEN",
            "fallback_activated": True
        })

        fallback_result = {
            "action": "Queue for retry",
            "message": "Payment processing queued for later",
            "retry_at": "2024-01-16T10:00:00Z"
        }

        self.state["current_step"] = 9
        return {
            "step": 9,
            "pillar": "cross_cutting",
            "action": "Circuit Breaker triggered",
            "result": "Billing service down - circuit OPEN - fallback activated",
            "failures": failures,
            "circuit_state": self.state["circuit_breaker_state"],
            "fallback": fallback_result
        }

    async def step_10_recovery(self) -> Dict:
        """
        Étape 10: Récupération après panne
        Pilier: 🛡️ Résilience (Recovery)
        """
        # Simulation récupération
        self.state["circuit_breaker_state"] = "HALF_OPEN"

        # Test avec une requête
        test_result = {"status": "OK", "latency_ms": 150}

        # Circuit fermé
        self.state["circuit_breaker_state"] = "CLOSED"
        self.state["recovery_performed"] = True

        # Traitement des messages en attente
        retry_result = {
            "queued_messages": 3,
            "processed": 3,
            "failed": 0
        }

        self.log("INFO", "Recovery completed - Circuit CLOSED", {
            "circuit_state": "CLOSED",
            "retry_result": retry_result
        })

        self.state["current_step"] = 10
        return {
            "step": 10,
            "pillar": "cross_cutting",
            "action": "Service recovery",
            "result": "Billing service recovered - queued messages processed",
            "circuit_transitions": ["OPEN", "HALF_OPEN", "CLOSED"],
            "retry_result": retry_result,
            "final_state": "All systems operational"
        }

    async def execute_all_steps(self, initial_data: Dict) -> List[Dict]:
        """Exécute toutes les étapes du scénario."""
        results = []

        # Étape 1: Gateway
        results.append(await self.step_1_receive_quote_request(initial_data))

        # Étape 2: Quote
        results.append(await self.step_2_calculate_quote(initial_data))

        # Étape 3: Policy
        results.append(await self.step_3_create_policy())

        # Étape 4: Publish Event
        results.append(await self.step_4_publish_policy_created())

        # Étape 5: Billing
        results.append(await self.step_5_billing_consumes())

        # Étape 6: Notifications
        results.append(await self.step_6_notifications_consumes())

        # Étape 7: CDC
        results.append(await self.step_7_cdc_sync())

        # Étape 8: Reporting
        results.append(await self.step_8_reporting_updated())

        # Étape 9: Failure
        results.append(await self.step_9_simulate_failure())

        # Étape 10: Recovery
        results.append(await self.step_10_recovery())

        return results

    def get_summary(self) -> Dict:
        """Retourne un résumé du scénario exécuté."""
        return {
            "scenario": "CROSS-04",
            "title": "Écosystème Complet",
            "trace_id": self.trace_id,
            "pillars_used": {
                "applications": ["Gateway", "BFF", "API Composition", "Saga"],
                "events": ["Pub/Sub", "Consumers (Billing, Notifications)"],
                "data": ["CDC", "ETL", "Reporting"],
                "cross_cutting": ["Circuit Breaker", "Retry", "Recovery"]
            },
            "state": {
                "quote_created": self.state["quote"] is not None,
                "policy_created": self.state["policy"] is not None,
                "events_published": len(self.state["events_published"]),
                "events_consumed": len(self.state["events_consumed"]),
                "dwh_synced": self.state["dwh_synced"],
                "recovery_performed": self.state["recovery_performed"]
            },
            "logs_count": len(self.logs)
        }


# Point d'entrée pour l'exécution depuis le sandbox
async def run_scenario(data: Dict = None) -> Dict:
    """Exécute le scénario CROSS-04."""
    scenario = Cross04Scenario()

    initial_data = data or {
        "customer_id": "C001",
        "product": "AUTO",
        "risk_data": {
            "vehicle_type": "sedan",
            "driver_age": 35
        }
    }

    results = await scenario.execute_all_steps(initial_data)

    return {
        "results": results,
        "summary": scenario.get_summary(),
        "logs": scenario.logs
    }


if __name__ == "__main__":
    # Test local
    result = asyncio.run(run_scenario())
    print(json.dumps(result, indent=2, default=str))
