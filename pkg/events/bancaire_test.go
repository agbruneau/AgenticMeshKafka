package events

import (
	"testing"
	"time"

	"github.com/hamba/avro/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompteOuvert_Serialization_Avro(t *testing.T) {
	schema, err := avro.Parse(CompteOuvertSchema)
	require.NoError(t, err, "schema should be valid")

	event := CompteOuvert{
		EventID:      "evt-123",
		Timestamp:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		CompteID:     "cpt-456",
		ClientID:     "cli-789",
		TypeCompte:   TypeCompteCOURANT,
		Devise:       "EUR",
		SoldeInitial: decimal.NewFromFloat(1000.50),
		Metadata:     nil,
	}

	// Serialize
	data, err := avro.Marshal(schema, event)
	require.NoError(t, err, "serialization should succeed")
	assert.NotEmpty(t, data, "serialized data should not be empty")
}

func TestCompteOuvert_Deserialization_Avro(t *testing.T) {
	schema, err := avro.Parse(CompteOuvertSchema)
	require.NoError(t, err, "schema should be valid")

	original := CompteOuvert{
		EventID:      "evt-123",
		Timestamp:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		CompteID:     "cpt-456",
		ClientID:     "cli-789",
		TypeCompte:   TypeCompteEPARGNE,
		Devise:       "EUR",
		SoldeInitial: decimal.NewFromFloat(2500.00),
		Metadata:     map[string]string{"source": "web"},
	}

	// Serialize
	data, err := avro.Marshal(schema, original)
	require.NoError(t, err, "serialization should succeed")

	// Deserialize
	var decoded CompteOuvert
	err = avro.Unmarshal(schema, data, &decoded)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, original.EventID, decoded.EventID)
	assert.Equal(t, original.CompteID, decoded.CompteID)
	assert.Equal(t, original.ClientID, decoded.ClientID)
	assert.Equal(t, original.TypeCompte, decoded.TypeCompte)
	assert.Equal(t, original.Devise, decoded.Devise)
	assert.True(t, original.SoldeInitial.Equal(decoded.SoldeInitial), "solde should match")
}

func TestDepotEffectue_RoundTrip(t *testing.T) {
	schema, err := avro.Parse(DepotEffectueSchema)
	require.NoError(t, err, "schema should be valid")

	original := DepotEffectue{
		EventID:   "evt-dep-001",
		Timestamp: time.Date(2024, 2, 20, 14, 45, 0, 0, time.UTC),
		CompteID:  "cpt-456",
		Montant:   decimal.NewFromFloat(500.75),
		Devise:    "EUR",
		Reference: "REF-2024-001",
		Canal:     CanalDepotGUICHET,
		Metadata:  nil,
	}

	// Serialize
	data, err := avro.Marshal(schema, original)
	require.NoError(t, err, "serialization should succeed")

	// Deserialize
	var decoded DepotEffectue
	err = avro.Unmarshal(schema, data, &decoded)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, original.EventID, decoded.EventID)
	assert.Equal(t, original.CompteID, decoded.CompteID)
	assert.True(t, original.Montant.Equal(decoded.Montant), "montant should match")
	assert.Equal(t, original.Reference, decoded.Reference)
	assert.Equal(t, original.Canal, decoded.Canal)
}

func TestVirementEmis_RoundTrip(t *testing.T) {
	schema, err := avro.Parse(VirementEmisSchema)
	require.NoError(t, err, "schema should be valid")

	original := VirementEmis{
		EventID:            "evt-vir-001",
		Timestamp:          time.Date(2024, 3, 10, 9, 15, 0, 0, time.UTC),
		CompteSourceID:     "cpt-source-001",
		CompteDestinationID: "cpt-dest-002",
		Montant:            decimal.NewFromFloat(1250.00),
		Devise:             "EUR",
		Motif:              "Paiement facture",
		Reference:          "VIR-2024-001",
		Statut:             StatutVirementINITIE,
		Metadata:           map[string]string{"priority": "normal"},
	}

	// Serialize
	data, err := avro.Marshal(schema, original)
	require.NoError(t, err, "serialization should succeed")

	// Deserialize
	var decoded VirementEmis
	err = avro.Unmarshal(schema, data, &decoded)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, original.EventID, decoded.EventID)
	assert.Equal(t, original.CompteSourceID, decoded.CompteSourceID)
	assert.Equal(t, original.CompteDestinationID, decoded.CompteDestinationID)
	assert.True(t, original.Montant.Equal(decoded.Montant), "montant should match")
	assert.Equal(t, original.Motif, decoded.Motif)
	assert.Equal(t, original.Reference, decoded.Reference)
	assert.Equal(t, original.Statut, decoded.Statut)
}

func TestTypeCompte_Values(t *testing.T) {
	assert.Equal(t, TypeCompte("COURANT"), TypeCompteCOURANT)
	assert.Equal(t, TypeCompte("EPARGNE"), TypeCompteEPARGNE)
	assert.Equal(t, TypeCompte("JOINT"), TypeCompteJOINT)
}

func TestCanalDepot_Values(t *testing.T) {
	assert.Equal(t, CanalDepot("GUICHET"), CanalDepotGUICHET)
	assert.Equal(t, CanalDepot("VIREMENT"), CanalDepotVIREMENT)
	assert.Equal(t, CanalDepot("CHEQUE"), CanalDepotCHEQUE)
	assert.Equal(t, CanalDepot("CARTE"), CanalDepotCARTE)
}

func TestStatutVirement_Values(t *testing.T) {
	assert.Equal(t, StatutVirement("INITIE"), StatutVirementINITIE)
	assert.Equal(t, StatutVirement("EN_COURS"), StatutVirementEN_COURS)
	assert.Equal(t, StatutVirement("COMPLETE"), StatutVirementCOMPLETE)
	assert.Equal(t, StatutVirement("REJETE"), StatutVirementREJETE)
}
