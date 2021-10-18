package types

import (
	"bytes"
	"testing"

	"github.com/celestiaorg/celestia-messages/payment/consts"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMountainRange(t *testing.T) {
	type test struct {
		l, k     uint64
		expected []uint64
	}
	tests := []test{
		{
			l:        11,
			k:        4,
			expected: []uint64{4, 4, 2, 1},
		},
		{
			l:        2,
			k:        64,
			expected: []uint64{2},
		},
		{ //should this test throw an error? we
			l:        64,
			k:        8,
			expected: []uint64{8, 8, 8, 8, 8, 8, 8, 8},
		},
	}
	for _, tt := range tests {
		res := powerOf2MountainRange(tt.l, tt.k)
		assert.Equal(t, tt.expected, res)
	}
}

func TestNextPowerOf2(t *testing.T) {
	type test struct {
		input    uint64
		expected uint64
	}
	tests := []test{
		{
			input:    2,
			expected: 2,
		},
		{
			input:    11,
			expected: 8,
		},
		{
			input:    511,
			expected: 256,
		},
		{
			input:    1,
			expected: 1,
		},
		{
			input:    0,
			expected: 0,
		},
	}
	for _, tt := range tests {
		res := nextPowerOf2(tt.input)
		assert.Equal(t, tt.expected, res)
	}
}

// TestCreateCommit only shows if something changed, it doesn't actually show
// the commit is being created correctly todo(evan): fix me.
func TestCreateCommitment(t *testing.T) {
	type test struct {
		k         uint64
		namespace []byte
		message   []byte
		expected  []byte
		expectErr bool
	}
	tests := []test{
		{
			k:         4,
			namespace: bytes.Repeat([]byte{0xFF}, 8),
			message:   bytes.Repeat([]byte{0xFF}, 11*consts.ShareSize),
			expected:  []byte{0x1c, 0x57, 0x89, 0x2f, 0xbe, 0xbf, 0xa2, 0xa4, 0x4c, 0x41, 0x9e, 0x2d, 0x88, 0xd5, 0x87, 0xc0, 0xbd, 0x37, 0xc0, 0x85, 0xbd, 0x10, 0x3c, 0x36, 0xd9, 0xa2, 0x4d, 0x4e, 0x31, 0xa2, 0xf8, 0x4e},
		},
		{
			k:         2,
			namespace: bytes.Repeat([]byte{0xFF}, 8),
			message:   bytes.Repeat([]byte{0xFF}, 100*consts.ShareSize),
			expectErr: true,
		},
	}
	for _, tt := range tests {
		res, err := CreateCommitment(tt.k, tt.namespace, tt.message)
		if tt.expectErr {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, res)
	}
}

func TestPadMessage(t *testing.T) {
	type test struct {
		input    []byte
		expected []byte
	}
	tests := []test{
		{
			input:    []byte{1},
			expected: append([]byte{1}, bytes.Repeat([]byte{0}, consts.ShareSize-1)...),
		},
		{
			input:    []byte{},
			expected: []byte{},
		},
		{
			input:    bytes.Repeat([]byte{1}, consts.ShareSize),
			expected: bytes.Repeat([]byte{1}, consts.ShareSize),
		},
		{
			input:    bytes.Repeat([]byte{1}, (3*consts.ShareSize)-10),
			expected: append(bytes.Repeat([]byte{1}, (3*consts.ShareSize)-10), bytes.Repeat([]byte{0}, 10)...),
		},
	}
	for _, tt := range tests {
		res := padMessage(tt.input)
		assert.Equal(t, tt.expected, res)
	}
}

func TestSignShareCommitments(t *testing.T) {
	type test struct {
		name    string
		ns, msg []byte
		ss      uint64
	}

	kb := generateKeyring(t, "test")

	signer := NewKeyringSigner(kb, "test", "chain-id")

	tests := []test{
		{
			name: "single share square size 2",
			ns:   []byte{1, 1, 1, 1, 1, 1, 1, 1},
			msg:  bytes.Repeat([]byte{1}, consts.ShareSize-8),
			ss:   2,
		},
		{
			name: "15 shares square size 4",
			ns:   []byte{1, 1, 1, 1, 1, 1, 1, 2},
			msg:  bytes.Repeat([]byte{2}, consts.ShareSize*15),
			ss:   4,
		},
	}

	for _, tt := range tests {
		wpfm, err := NewWirePayForMessage(tt.ns, tt.msg, tt.ss)
		require.NoError(t, err, tt.name)
		err = wpfm.SignShareCommitments(signer, signer.NewTxBuilder())
		// there should be no error
		assert.NoError(t, err)
		// the signature should exist
		assert.Equal(t, len(wpfm.MessageShareCommitment[0].Signature), 64)

		// verify the signature
		unsignedPFM, err := wpfm.unsignedPayForMessage(tt.ss)
		require.NoError(t, err)
		tx, err := signer.BuildSignedTx(signer.NewTxBuilder(), unsignedPFM)
		require.NoError(t, err)

		// Generate the bytes to be signed.
		bytesToSign, err := signer.encCfg.TxConfig.SignModeHandler().GetSignBytes(
			signing.SignMode_SIGN_MODE_DIRECT,
			authsigning.SignerData{
				ChainID:       signer.chainID,
				AccountNumber: signer.accountNumber,
				Sequence:      signer.sequence,
			},
			tx,
		)
		require.NoError(t, err)

		// verify the signature using the public key
		assert.True(t, signer.GetSignerInfo().GetPubKey().VerifySignature(
			bytesToSign,
			wpfm.MessageShareCommitment[0].Signature,
		))

	}
}

func TestWirePayForMessage_ValidateBasic(t *testing.T) {
	type test struct {
		name      string
		msg       *MsgWirePayForMessage
		expectErr bool
		errStr    string
	}

	// valid pfm
	validMsg := validWirePayForMessage(t)

	// pfm with bad ns id
	badIDMsg := validWirePayForMessage(t)
	badIDMsg.MessageNameSpaceId = []byte{1, 2, 3, 4, 5, 6, 7}

	// pfm that uses reserved ns id
	reservedMsg := validWirePayForMessage(t)
	reservedMsg.MessageNameSpaceId = []byte{0, 0, 0, 0, 0, 0, 0, 100}

	// pfm that has a wrong msg size
	invalidMsgSizeMsg := validWirePayForMessage(t)
	invalidMsgSizeMsg.Message = bytes.Repeat([]byte{1}, consts.ShareSize-20)

	// pfm that has a wrong msg size
	invalidDeclaredMsgSizeMsg := validWirePayForMessage(t)
	invalidDeclaredMsgSizeMsg.MessageSize = 999

	// pfm with bad commitment
	badCommitMsg := validWirePayForMessage(t)
	badCommitMsg.MessageShareCommitment[0].ShareCommitment = []byte{1, 2, 3, 4}

	tests := []test{
		{
			name: "valid msg",
			msg:  validMsg,
		},
		{
			name:      "bad ns ID",
			msg:       badIDMsg,
			expectErr: true,
			errStr:    "invalid namespace length",
		},
		{
			name:      "reserved ns id",
			msg:       reservedMsg,
			expectErr: true,
			errStr:    "uses a reserved namesapce ID",
		},
		{
			name:      "invalid msg size",
			msg:       invalidMsgSizeMsg,
			expectErr: true,
			errStr:    "Share message must be divisible",
		},
		{
			name:      "bad declared message size",
			msg:       invalidDeclaredMsgSizeMsg,
			expectErr: true,
			errStr:    "Declared Message size does not match actual Message size",
		},
		{
			name:      "bad commitment",
			msg:       badCommitMsg,
			expectErr: true,
			errStr:    "invalid commit for square size",
		},
	}

	for _, tt := range tests {
		err := tt.msg.ValidateBasic()
		if tt.expectErr {
			require.NotNil(t, err, tt.name)
			require.Contains(t, err.Error(), tt.errStr, tt.name)
			continue
		}
		require.NoError(t, err, tt.name)
	}
}

func validWirePayForMessage(t *testing.T) *MsgWirePayForMessage {
	msg, err := NewWirePayForMessage(
		[]byte{1, 2, 3, 4, 5, 6, 7, 8},
		bytes.Repeat([]byte{1}, 1000),
		16, 32, 64,
	)
	if err != nil {
		panic(err)
	}

	signer := generateKeyringSigner(t)

	err = msg.SignShareCommitments(signer, signer.NewTxBuilder())
	if err != nil {
		panic(err)
	}
	return msg
}
