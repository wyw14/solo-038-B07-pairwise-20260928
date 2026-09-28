package dryer

import (
	"math"
	"testing"
	"time"

	"github.com/wyw14/solo-038/internal/model"
)

func testEvidence(t *testing.T) (model.Job, model.Interval) {
	t.Helper()
	now := time.Now()
	job := model.NewJob(now)
	job.RollGeneration = 1
	interval := model.Interval{
		JobID:          job.ID,
		JobGeneration:  job.Generation,
		RollGeneration: job.RollGeneration,
		StartMeter:     0,
		EndMeter:       10,
		OpenedAt:       now,
		ClosedAt:       now,
	}
	return job, interval
}

func TestBuildProofRejectsInvalidReadings(t *testing.T) {
	job, interval := testEvidence(t)
	now := time.Now()
	cases := map[string]struct {
		temperature float64
		airflow     float64
	}{
		"nan temperature":     {math.NaN(), 24},
		"positive inf temp":   {math.Inf(1), 24},
		"negative inf temp":   {math.Inf(-1), 24},
		"nan airflow":         {100, math.NaN()},
		"positive inf airflow": {100, math.Inf(1)},
		"zero temperature":    {0, 24},
		"negative airflow":    {100, -1},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := BuildProof(job, interval, tc.temperature, tc.airflow, now); err == nil {
				t.Fatalf("expected error for %s, proof was generated", name)
			}
		})
	}
}

func TestBuildProofAndCoversWithValidReadings(t *testing.T) {
	job, interval := testEvidence(t)
	now := time.Now()
	proof, err := BuildProof(job, interval, 100, 24, now)
	if err != nil {
		t.Fatalf("valid readings should build proof: %v", err)
	}
	if !proof.Covers(interval) {
		t.Fatal("proof from valid readings should cover the interval")
	}
	narrower := model.Interval{
		JobID:          interval.JobID,
		JobGeneration:  interval.JobGeneration,
		RollGeneration: interval.RollGeneration,
		StartMeter:     2,
		EndMeter:       8,
		OpenedAt:       interval.OpenedAt,
		ClosedAt:       interval.ClosedAt,
	}
	if !proof.Covers(narrower) {
		t.Fatal("proof should cover a narrower interval")
	}
	outside := interval
	outside.StartMeter = 5
	outside.EndMeter = 20
	if proof.Covers(outside) {
		t.Fatal("proof must not cover an interval beyond its range")
	}
}

func TestApplyRejectsNonFiniteTemperature(t *testing.T) {
	state := NewState()
	now := time.Now()
	job, _ := testEvidence(t)
	if err := state.Start(string(job.ID), job.Generation, 100, "production", now); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := state.Apply(string(job.ID), job.Generation, math.NaN(), now); err == nil {
		t.Fatal("NaN temperature must be rejected")
	}
	if err := state.Apply(string(job.ID), job.Generation, math.Inf(1), now); err == nil {
		t.Fatal("+Inf temperature must be rejected")
	}
	if err := state.Apply(string(job.ID), job.Generation, 95, now); err != nil {
		t.Fatalf("finite temperature should be accepted: %v", err)
	}
}

func TestStartRejectsNonFiniteTarget(t *testing.T) {
	state := NewState()
	now := time.Now()
	job, _ := testEvidence(t)
	if err := state.Start(string(job.ID), job.Generation, math.NaN(), "production", now); err == nil {
		t.Fatal("NaN target must be rejected")
	}
}
