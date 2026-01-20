// Package events provides Go types for Avro-serialized banking domain events.
package events

import (
	"time"

	"github.com/shopspring/decimal"
)

// TypeCompte represents the type of bank account
type TypeCompte string

const (
	TypeCompteCOURANT TypeCompte = "COURANT"
	TypeCompteEPARGNE TypeCompte = "EPARGNE"
	TypeCompteJOINT   TypeCompte = "JOINT"
)

// CanalDepot represents the channel through which a deposit was made
type CanalDepot string

const (
	CanalDepotGUICHET  CanalDepot = "GUICHET"
	CanalDepotVIREMENT CanalDepot = "VIREMENT"
	CanalDepotCHEQUE   CanalDepot = "CHEQUE"
	CanalDepotCARTE    CanalDepot = "CARTE"
)

// StatutVirement represents the status of a wire transfer
type StatutVirement string

const (
	StatutVirementINITIE   StatutVirement = "INITIE"
	StatutVirementEN_COURS StatutVirement = "EN_COURS"
	StatutVirementCOMPLETE StatutVirement = "COMPLETE"
	StatutVirementREJETE   StatutVirement = "REJETE"
)

// CompteOuvert represents an account opening event
type CompteOuvert struct {
	EventID      string            `avro:"event_id" json:"event_id"`
	Timestamp    time.Time         `avro:"timestamp" json:"timestamp"`
	CompteID     string            `avro:"compte_id" json:"compte_id"`
	ClientID     string            `avro:"client_id" json:"client_id"`
	TypeCompte   TypeCompte        `avro:"type_compte" json:"type_compte"`
	Devise       string            `avro:"devise" json:"devise"`
	SoldeInitial decimal.Decimal   `avro:"solde_initial" json:"solde_initial"`
	Metadata     map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// DepotEffectue represents a deposit event
type DepotEffectue struct {
	EventID   string            `avro:"event_id" json:"event_id"`
	Timestamp time.Time         `avro:"timestamp" json:"timestamp"`
	CompteID  string            `avro:"compte_id" json:"compte_id"`
	Montant   decimal.Decimal   `avro:"montant" json:"montant"`
	Devise    string            `avro:"devise" json:"devise"`
	Reference string            `avro:"reference" json:"reference"`
	Canal     CanalDepot        `avro:"canal" json:"canal"`
	Metadata  map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// VirementEmis represents a wire transfer event
type VirementEmis struct {
	EventID             string            `avro:"event_id" json:"event_id"`
	Timestamp           time.Time         `avro:"timestamp" json:"timestamp"`
	CompteSourceID      string            `avro:"compte_source_id" json:"compte_source_id"`
	CompteDestinationID string            `avro:"compte_destination_id" json:"compte_destination_id"`
	Montant             decimal.Decimal   `avro:"montant" json:"montant"`
	Devise              string            `avro:"devise" json:"devise"`
	Motif               string            `avro:"motif" json:"motif"`
	Reference           string            `avro:"reference" json:"reference"`
	Statut              StatutVirement    `avro:"statut" json:"statut"`
	Metadata            map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// CompteOuvertSchema is the Avro schema for CompteOuvert events
const CompteOuvertSchema = `{
  "type": "record",
  "name": "CompteOuvert",
  "namespace": "com.edalab.bancaire.events",
  "doc": "Événement émis lors de l'ouverture d'un compte bancaire",
  "fields": [
    {"name": "event_id", "type": "string"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "compte_id", "type": "string"},
    {"name": "client_id", "type": "string"},
    {"name": "type_compte", "type": {"type": "enum", "name": "TypeCompte", "symbols": ["COURANT", "EPARGNE", "JOINT"]}},
    {"name": "devise", "type": "string", "default": "EUR"},
    {"name": "solde_initial", "type": {"type": "bytes", "logicalType": "decimal", "precision": 18, "scale": 2}},
    {"name": "metadata", "type": ["null", {"type": "map", "values": "string"}], "default": null}
  ]
}`

// DepotEffectueSchema is the Avro schema for DepotEffectue events
const DepotEffectueSchema = `{
  "type": "record",
  "name": "DepotEffectue",
  "namespace": "com.edalab.bancaire.events",
  "doc": "Événement émis lors d'un dépôt sur un compte bancaire",
  "fields": [
    {"name": "event_id", "type": "string"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "compte_id", "type": "string"},
    {"name": "montant", "type": {"type": "bytes", "logicalType": "decimal", "precision": 18, "scale": 2}},
    {"name": "devise", "type": "string", "default": "EUR"},
    {"name": "reference", "type": "string"},
    {"name": "canal", "type": {"type": "enum", "name": "CanalDepot", "symbols": ["GUICHET", "VIREMENT", "CHEQUE", "CARTE"]}},
    {"name": "metadata", "type": ["null", {"type": "map", "values": "string"}], "default": null}
  ]
}`

// VirementEmisSchema is the Avro schema for VirementEmis events
const VirementEmisSchema = `{
  "type": "record",
  "name": "VirementEmis",
  "namespace": "com.edalab.bancaire.events",
  "doc": "Événement émis lors de l'émission d'un virement bancaire",
  "fields": [
    {"name": "event_id", "type": "string"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "compte_source_id", "type": "string"},
    {"name": "compte_destination_id", "type": "string"},
    {"name": "montant", "type": {"type": "bytes", "logicalType": "decimal", "precision": 18, "scale": 2}},
    {"name": "devise", "type": "string", "default": "EUR"},
    {"name": "motif", "type": "string"},
    {"name": "reference", "type": "string"},
    {"name": "statut", "type": {"type": "enum", "name": "StatutVirement", "symbols": ["INITIE", "EN_COURS", "COMPLETE", "REJETE"]}},
    {"name": "metadata", "type": ["null", {"type": "map", "values": "string"}], "default": null}
  ]
}`
