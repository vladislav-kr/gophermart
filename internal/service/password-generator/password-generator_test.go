package passwordgenerator

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func Test_generator(t *testing.T) {
	passwordTest := "QwErFDJD3236$FJ2324fjf1231"

	passGen := New(bcrypt.DefaultCost)

	hash, err := passGen.GenerateFromPassword([]byte(passwordTest))
	require.NoError(t, err)

	err = passGen.CompareHashAndPassword(hash, []byte(passwordTest))
	require.NoError(t, err)

	err = passGen.CompareHashAndPassword(hash, []byte("QwErFDJD3236$FJ2324fjf1232"))
	require.Error(t, err)

}
