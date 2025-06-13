package usecase_test

import (
	"firestore-clone/internal/firestore/usecase"
	"testing"
)

// TestFirestoreUsecaseInterface_Contract checks that FirestoreUsecase implements the interface contract.
func TestFirestoreUsecaseInterface_Contract(t *testing.T) {
	var _ usecase.FirestoreUsecaseInterface = &usecase.FirestoreUsecase{}
}
