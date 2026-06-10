package services

import "testing"

func TestPetBalanceFeatureServiceComputeDebtSubsidy(t *testing.T) {
	params := DebtSubsidyParams{Enabled: true, SubsidyRate: 0.18}
	got := PetBalanceFeatureService.ComputeDebtSubsidy(-500, params)
	if got != 90 {
		t.Fatalf("ComputeDebtSubsidy(-500) = %d, want 90", got)
	}
}

func TestPetBalanceFeatureServiceComputeDebtSubsidyLightning(t *testing.T) {
	params := DebtSubsidyParams{Enabled: true, SubsidyRate: 0.20}
	got := PetBalanceFeatureService.ComputeDebtSubsidy(-200, params)
	if got != 40 {
		t.Fatalf("ComputeDebtSubsidy(-200) = %d, want 40", got)
	}
}

func TestPetBalanceFeatureServiceComputeDepositInterest(t *testing.T) {
	params := DepositInterestParams{Enabled: true, InterestRate: 0.05, CapPerDay: 1000}
	got := PetBalanceFeatureService.ComputeDepositInterest(10000, params)
	if got != 500 {
		t.Fatalf("ComputeDepositInterest(10000) = %d, want 500", got)
	}
}

func TestPetBalanceFeatureServiceComputeDepositInterestCap(t *testing.T) {
	params := DepositInterestParams{Enabled: true, InterestRate: 0.05, CapPerDay: 1000}
	got := PetBalanceFeatureService.ComputeDepositInterest(25000, params)
	if got != 1000 {
		t.Fatalf("ComputeDepositInterest(25000) = %d, want 1000", got)
	}
}

func TestPetBalanceFeatureServiceCanSpendWithDebtFloor(t *testing.T) {
	if !PetBalanceFeatureService.CanSpend(100, 350, -300) {
		t.Fatalf("expected spend to be allowed at debt floor")
	}
	if PetBalanceFeatureService.CanSpend(100, 401, -300) {
		t.Fatalf("expected spend to be rejected below debt floor")
	}
}
