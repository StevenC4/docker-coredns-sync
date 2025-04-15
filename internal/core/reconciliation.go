package core

import (
	"github.com/StevenC4/docker-coredns-sync/internal/dns"
	"github.com/StevenC4/docker-coredns-sync/internal/intent"
	"github.com/rs/zerolog"
)

func shouldReplaceExisting(new, existing *intent.RecordIntent, logger zerolog.Logger) bool {
	l := logger.With().Str("reconciler", "filter").Str("new_record", new.Render()).Str("existing_record", existing.Render()).Logger()
	if new.Force && !existing.Force {
		l.Trace().Msg("Replacing existing record due to force label on new record")
		return true
	} else if !new.Force && existing.Force {
		l.Trace().Msg("Keeping existing record due to force label on existing record")
		return false
	} else if new.Created.Before(existing.Created) {
		l.Trace().Msg("Replacing existing record due to the new record's container being older")
		return true
	}
	l.Trace().Msg("Keeping existing record due to the existing record's container being older")
	return false
}

func shouldReplaceAllExisting(new *intent.RecordIntent, existing []*intent.RecordIntent, logger zerolog.Logger) bool {
	// Returns True if `new` (CNAME) should replace all `existing` (A records).

	// Rules:
	// - If any existing is force and new is not, new loses.
	// - If new is force and all existing are not, new wins.
	// - If mixed force values exist and new is force:
	//     - New must be older than *all* existing force records.
	//     - Otherwise, new loses.
	// - If force flags match for all (either all force or all non-force), the oldest record wins.
	existingRecordStrings := make([]string, 0, len(existing))
	for _, ri := range existing {
		existingRecordStrings = append(existingRecordStrings, ri.Render())
	}
	l := logger.With().Str("reconciler", "filter").Str("new_record", new.Render()).Strs("existing_records", existingRecordStrings).Logger()
	if len(existing) == 0 {
		return true
	}

	anyForce := false
	allForce := true
	allNonForce := true
	newCreatedBeforeAllOldWithForce := true
	newCreatedBeforeAll := true
	for _, ri := range existing {
		newCreatedBeforeExisting := new.Created.Before(ri.Created)
		if ri.Force {
			anyForce = true
			allNonForce = false
			if !newCreatedBeforeExisting {
				newCreatedBeforeAllOldWithForce = false
			}
		} else {
			allForce = false
		}
		if !newCreatedBeforeExisting {
			newCreatedBeforeAll = false
		}
	}

	// If any existing is force and new is not, new loses.
	if anyForce && !new.Force {
		l.Trace().Msg("Keeping all existing records because one of their containers has the force label and the new record's container does not")
		return false
	}

	// If new is force and all existing are not, new wins.
	if new.Force && allNonForce {
		l.Trace().Msg("Replacing all existing records with the new one because none of the existing record's containers has the force label and the new record's container does")
		return true
	}

	// If mixed force values exist and new is force:
	// New must be older than all existing force records.
	if new.Force && !allForce {
		if newCreatedBeforeAllOldWithForce {
			l.Trace().Msg("Replacing all existing new records with the new one because the new record's container has the force label and was created before all of the existing records' containers that have the force label")
			return true
		}
		l.Trace().Msg("Keeping all existing new records - although the new one's container has the force label, one or more of the existing records' containers with the force label was created before it")
		return false
	}

	// Otherwise, when force flags match (either all true or all false), the oldest wins.
	if newCreatedBeforeAll {
		l.Trace().Msg("Replacing all existing records with the new one because none of the containers have the force label and the new record's container is older than the containers of all the existing records")
		return true
	}
	l.Trace().Msg("Keeping all existing records because none of the containers have the force label and the new record's container is not older than the containers of all the existing records")
	return false
}

// FilterRecordIntents receives a slice of RecordIntent (desired) and filters out conflicting ones.
// It returns a reconciled slice of RecordIntent.
func FilterRecordIntents(recordIntents []*intent.RecordIntent, logger zerolog.Logger) []*intent.RecordIntent {
	logger.Debug().Msg("Reconciling desired records against each other")

	desiredByNameType := NewNestedRecordMap()

	for _, ri := range recordIntents {
		record := ri.Record
		name := record.GetName()
		value := record.GetValue()

		existingARecordIntents, hasARecords := desiredByNameType.PeekNameTypeRecords(name, "A")
		existingCNAMERecordIntents, hasCNAMERecords := desiredByNameType.PeekNameTypeRecords(name, "CNAME")

		// Check for conflicts between A and CNAME record types:
		// We want to enforce: if an A record exists, and a CNAME comes in for same name, we choose one based on business rules.
		if _, ok := record.(*dns.ARecord); ok {
			if hasCNAMERecords {
				// Get an existing CNAME record (assume only one exists)
				existingCNAMERecordIntent := existingCNAMERecordIntents[0]
				if shouldReplaceExisting(ri, existingCNAMERecordIntent, logger) {
					// Remove CNAME record and add A record
					desiredByNameType.DeleteNameType(name, "CNAME")
					desiredByNameType.Get(name).Get("A").Set(value, ri)
				}
			} else if existingARecordIntent, exists := desiredByNameType.PeekNameTypeRecord(name, "A", value); exists {
				if shouldReplaceExisting(ri, existingARecordIntent, logger) {
					// Replace A record
					desiredByNameType.Get(name).Get("A").Set(value, ri)
				}
			} else {
				// No conflict - just add it
				desiredByNameType.Get(name).Get("A").Set(value, ri)
			}
		} else if _, ok := record.(*dns.CNAMERecord); ok {
			if hasARecords {
				// Get existing A records (more than one may exist)
				if shouldReplaceAllExisting(ri, existingARecordIntents, logger) {
					// Remoev all A records with the name and add a CNAME record
					desiredByNameType.DeleteNameType(name, "A")
					desiredByNameType.Get(name).Get("CNAME").Set(value, ri)
				}
			} else if hasCNAMERecords {
				// Get existing CNAME record - assume only one
				existingCNAMERecordIntent := existingCNAMERecordIntents[0]
				if shouldReplaceExisting(ri, existingCNAMERecordIntent, logger) {
					// Replace CNAME record
					desiredByNameType.Get(name).Get("CNAME").Set(value, ri)
				}
			} else {
				// No conflict - just add it
				desiredByNameType.Get(name).Get("CNAME").Set(value, ri)
			}
		}
	}

	return desiredByNameType.GetAllValues()
}

func ReconcileAndValidate(desired, actual []*intent.RecordIntent, logger zerolog.Logger) ([]*intent.RecordIntent, []*intent.RecordIntent) {
	toAddMap := map[string]*intent.RecordIntent{}
	toRemoveMap := map[string]*intent.RecordIntent{}

	actualByNameType := NewNestedRecordMap()
	desiredSet := make(map[string]struct{}, len(desired))
	for _, ri := range desired {
		desiredSet[ri.Key()] = struct{}{}
	}

	// Step 1: Remove stale records and build lookup structure
	for _, ri := range actual {
		if _, exists := desiredSet[ri.Key()]; !exists {
			logger.Info().Msgf("[reconciler] Removing stale record: %s (owned by %s/%s)",
				ri.Record.Render(), ri.Hostname, ri.ContainerName)
			toRemoveMap[ri.Key()] = ri
		} else {
			name := ri.Record.GetName()
			recordType := ri.Record.GetType()
			value := ri.Record.GetValue()
			actualByNameType.Get(name).Get(recordType).Set(value, ri)
		}
	}

	// Step 2: Reconcile each desired record
	for _, desiredRecordIntent := range desired {
		name := desiredRecordIntent.Record.GetName()
		value := desiredRecordIntent.Record.GetValue()

		evictions := map[string]*intent.RecordIntent{}

		if _, ok := desiredRecordIntent.Record.(*dns.ARecord); ok {
			if actualRecordIntents, exists := actualByNameType.PeekNameTypeRecords(name, "CNAME"); exists {
				// Conflict: desired A, actual has CNAME(s)
				actualRecordIntent := actualRecordIntents[0]
				if desiredRecordIntent.Force {
					actualCnameStrings := make([]string, len(actualRecordIntents))
					for i, ri := range actualRecordIntents {
						actualCnameStrings[i] = ri.Render()
						evictions[ri.Key()] = ri
					}
					logger.Warn().Strs("actual_record_intents", actualCnameStrings).Str("desired_record_intent", desiredRecordIntent.Render()).Msg("Record conflict between local and remote - evicting remote due to force container label")
				} else if desiredRecordIntent.Created.Before(actualRecordIntent.Created) {
					actualCnameStrings := make([]string, len(actualRecordIntents))
					for i, ri := range actualRecordIntents {
						actualCnameStrings[i] = ri.Render()
						evictions[ri.Key()] = ri
					}
					logger.Warn().Strs("actual_record_intents", actualCnameStrings).Str("desired_record_intent", desiredRecordIntent.Render()).Msg("Record conflict between local and remote - evicting remote due to container age")
				} else {
					// We're not evicting, so skip the rest for this record
					continue
				}
			} else if actualRecordIntent, exists := actualByNameType.PeekNameTypeRecord(name, "A", value); exists {
				if actualRecordIntent.Equal(*desiredRecordIntent) {
					// Skip - we don't need to replace ourselves
					continue
				} else if desiredRecordIntent.Force {
					logger.Warn().Str("actual_record_intent", actualRecordIntent.Render()).Str("desired_record_intent", desiredRecordIntent.Render()).Msg("Record conflict between local and remote - evicting remote due to force container label")
					evictions[actualRecordIntent.Key()] = actualRecordIntent
				} else if desiredRecordIntent.Created.Before(actualRecordIntent.Created) {
					logger.Warn().Str("actual_record_intent", actualRecordIntent.Render()).Str("desired_record_intent", desiredRecordIntent.Render()).Msg("Record conflict between local and remote - evicting remote due to container age")
					evictions[actualRecordIntent.Key()] = actualRecordIntent
				} else {
					// We're not evicting, so skip the rest for this record
					continue
				}
			}
			// Else: don't skip - just add with no evictions - no need for an else statement, this will just work
		} else if _, ok := desiredRecordIntent.Record.(*dns.CNAMERecord); ok {
			if actualRecordIntents, exists := actualByNameType.PeekNameTypeRecords(name, "A"); exists {
				desiredOlderThanAllActual := true
				for _, ri := range actualRecordIntents {
					desiredOlderThanAllActual = desiredOlderThanAllActual && desiredRecordIntent.Created.Before(ri.Created)
				}

				if desiredRecordIntent.Force {
					actualAStrings := make([]string, len(actualRecordIntents))
					for i, ri := range actualRecordIntents {
						actualAStrings[i] = ri.Render()
						evictions[ri.Key()] = ri
					}
					logger.Warn().Strs("actual_record_intents", actualAStrings).Str("desired_record_intent", desiredRecordIntent.Render()).Msg("Record conflict between local and remote - evicting remote due to force container label")
				} else if desiredOlderThanAllActual {
					actualAStrings := make([]string, len(actualRecordIntents))
					for i, ri := range actualRecordIntents {
						actualAStrings[i] = ri.Render()
						evictions[ri.Key()] = ri
					}
					logger.Warn().Strs("actual_record_intents", actualAStrings).Str("desired_record_intent", desiredRecordIntent.Render()).Msg("Record conflict between local and remote - evicting remote due to container age")
				} else {
					continue
				}
			} else if actualRecordIntents, exists := actualByNameType.PeekNameTypeRecords(name, "CNAME"); exists {
				actualRecordIntent := actualRecordIntents[0]
				if actualRecordIntent.Equal(*desiredRecordIntent) {
					// Skip - we don't need to replace ourselves
					continue
				} else if desiredRecordIntent.Force {
					actualCnameStrings := make([]string, len(actualRecordIntents))
					for i, ri := range actualRecordIntents {
						actualCnameStrings[i] = ri.Render()
						evictions[ri.Key()] = ri
					}
					logger.Warn().Strs("actual_record_intents", actualCnameStrings).Str("desired_record_intent", desiredRecordIntent.Render()).Msg("Record conflict between local and remote - evicting remote due to force container label")
				} else if desiredRecordIntent.Created.Before(actualRecordIntent.Created) {
					actualCnameStrings := make([]string, len(actualRecordIntents))
					for i, ri := range actualRecordIntents {
						actualCnameStrings[i] = ri.Render()
						evictions[ri.Key()] = ri
					}
					logger.Warn().Strs("actual_record_intents", actualCnameStrings).Str("desired_record_intent", desiredRecordIntent.Render()).Msg("Record conflict between local and remote - evicting remote due to container age")
				} else {
					continue
				}
			}
			// Else: don't skip - just add with no evictions - no need for an else statement, this will just work
		}

		// Step 3: Simulate state for validation
		keysToRemove := make(map[string]struct{})
		for key := range toRemoveMap {
			keysToRemove[key] = struct{}{}
		}
		for key := range evictions {
			keysToRemove[key] = struct{}{}
		}
		var simulated []*intent.RecordIntent
		for _, ri := range actual {
			if _, exists := keysToRemove[ri.Key()]; !exists {
				simulated = append(simulated, ri)
			}
		}

		// Step 4: Validate and commit
		if err := ValidateRecord(desiredRecordIntent, simulated, logger); err == nil {
			logger.Info().Msgf("[reconciler] Adding new record: %s", desiredRecordIntent.Render())
			toAddMap[desiredRecordIntent.Record.Key()] = desiredRecordIntent
			for k, v := range evictions {
				toRemoveMap[k] = v
			}
		} else {
			logger.Warn().Err(err).Msgf("[reconciler] Skipping invalid record %s", desiredRecordIntent.Record.Render())
		}
	}

	// Step 5: Convert maps to slices
	var toAdd, toRemove []*intent.RecordIntent
	for _, r := range toAddMap {
		toAdd = append(toAdd, r)
	}
	for _, r := range toRemoveMap {
		toRemove = append(toRemove, r)
	}
	return toAdd, toRemove
}
