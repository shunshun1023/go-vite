package ledger

import (
	"math/big"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/crypto"
	"bytes"
	"github.com/vitelabs/go-vite/vitepb"
)

type AccountBlockMeta struct {
	// Account id
	AccountId *big.Int

	// AccountBlock height
	Height *big.Int

	// Block status, 0 means unknow, 1 means open, 2 means closed
	Status int

	// Is snapshotted
	IsSnapshotted bool
}

func (abm *AccountBlockMeta) NetSerialize () ([]byte, error) {
	return abm.DbSerialize()
}

func (abm *AccountBlockMeta) NetDeserialize (buf []byte) (error) {
	return abm.DbDeserialize(buf)
}

func (abm *AccountBlockMeta) DbSerialize () ([]byte, error) {
	accountBlockMetaPb := &vitepb.AccountBlockMeta{
		AccountId: abm.AccountId.Bytes(),
		Height: abm.Height.Bytes(),
		Status: uint32(abm.Status),
		IsSnapshotted: abm.IsSnapshotted,
	}

	return proto.Marshal(accountBlockMetaPb)
}

func (abm *AccountBlockMeta) DbDeserialize (buf []byte) (error) {
	accountBlockMetaPb := &vitepb.AccountBlockMeta{}
	if err := proto.Unmarshal(buf, accountBlockMetaPb); err != nil {
		return err
	}

	abm.AccountId = &big.Int{}
	abm.AccountId.SetBytes(accountBlockMetaPb.AccountId)

	abm.Height = &big.Int{}
	abm.Height.SetBytes(accountBlockMetaPb.Height)

	abm.Status = int(accountBlockMetaPb.Status)
	abm.IsSnapshotted = accountBlockMetaPb.IsSnapshotted

	return nil
}

type AccountBlockList []*AccountBlock

func (ablist AccountBlockList) NetSerialize () ([]byte, error) {
	accountBlockListNetPB := &vitepb.AccountBlockListNet{}
	accountBlockListNetPB.Blocks = []*vitepb.AccountBlockNet{}

	for _, accountBlock := range ablist {
		accountBlockListNetPB.Blocks = append(accountBlockListNetPB.Blocks, accountBlock.GetNetPB())
	}
	return proto.Marshal(accountBlockListNetPB)
}

func (ablist AccountBlockList) NetDeserialize (buf []byte) (error) {
	accountBlockListNetPB := &vitepb.AccountBlockListNet{}
	if err := proto.Unmarshal(buf, accountBlockListNetPB); err != nil {
		return err
	}

	for _, blockPB := range accountBlockListNetPB.Blocks {
		block := &AccountBlock{}
		block.SetByNetPB(blockPB)
		ablist = append(ablist, block)
	}

	return nil
}

type AccountBlock struct {
	// [Optional] AccountBlockMeta
	Meta *AccountBlockMeta

	// Self account
	AccountAddress *types.Address

	// Public key
	PublicKey ed25519.PublicKey

	// Receiver account, exists in send block
	To *types.Address

	// [Optional] Sender account, exists in receive block
	From *types.Address

	// Correlative send block hash, exists in receive block
	FromHash *types.Hash

	// Last block hash
	PrevHash *types.Hash

	// Block hash
	Hash *types.Hash

	// Balance of current account
	Balance *big.Int

	// Amount of this transaction
	Amount *big.Int

	// Timestamp
	Timestamp uint64

	// Id of token received or sent
	TokenId *types.TokenTypeId

	// [Optional] Height of last transaction block in this token
	LastBlockHeightInToken *big.Int

	// Data requested or repsonsed
	Data string

	// Snapshot timestamp
	SnapshotTimestamp *types.Hash

	// Signature of current block
	Signature []byte

	// PoW nounce
	Nounce []byte

	// PoW difficulty
	Difficulty []byte

	// Service fee
	FAmount *big.Int
}

func (ab *AccountBlock) SetHash () error {
	// Hash source data:
	// PrevHash|Height|AccountAddress|PublicKey|To or FromHash|Timestamp|TokenId|Amount|Data|SnapshotTimestamp|Nounce|Difficulty|FAmount
	var source []byte
	source = append(source, ab.PrevHash.Bytes()...)
	source = append(source, []byte(ab.Meta.Height.String())...)
	source = append(source, ab.AccountAddress.Bytes()...)
	source = append(source, ab.PublicKey...)

	if ab.To != nil {
		source = append(source, ab.To.Bytes()...)
	} else {
		source = append(source, ab.FromHash.Bytes()...)
	}

	source = append(source, []byte(string(ab.Timestamp))...)
	source = append(source, ab.TokenId.Bytes()...)
	source = append(source, []byte(ab.Amount.String())...)
	source = append(source, []byte(ab.Data)...)
	source = append(source, ab.SnapshotTimestamp.Bytes()...)

	source = append(source, ab.Nounce...)
	source = append(source, ab.Difficulty...)
	source = append(source, []byte(ab.FAmount.String())...)

	hash, err := types.BytesToHash(crypto.Hash(len(source), source))
	if err != nil {
		return err
	}

	ab.Hash = &hash
	return nil
}

// Genesis block
func (ab *AccountBlock) IsGenesisBlock () bool {
	return ab.IsSendBlock() && bytes.Equal(ab.AccountAddress.Bytes(), GenesisAccount.Bytes()) && ab.PrevHash == nil
}

// Send block
func (ab *AccountBlock) IsSendBlock () bool {
	return ab.To != nil
}

// Receive block
func (ab *AccountBlock) IsReceiveBlock () bool {
	return ab.FromHash != nil
}

// Mintage block
func (ab * AccountBlock) IsMintageBlock () bool {
	return ab.IsSendBlock() && bytes.Equal(ab.To.Bytes(), MintageAddress.Bytes())
}

func (ab *AccountBlock) GetNetPB () *vitepb.AccountBlockNet {
	accountBlockNetPB := &vitepb.AccountBlockNet{
		Data: ab.Data,
		Timestamp: ab.Timestamp,

		Signature: ab.Signature,
		Nounce: ab.Nounce,

		Difficulty: ab.Difficulty,

	}
	if ab.Meta != nil {
		accountBlockNetPB.Meta = &vitepb.AccountBlockMeta{
			AccountId: ab.Meta.AccountId.Bytes(),
			Height: ab.Meta.Height.Bytes(),
			Status: uint32(ab.Meta.Status),
		}
	}
	if ab.To != nil {
		accountBlockNetPB.To = ab.To.Bytes()
	}

	if ab.PrevHash != nil {
		accountBlockNetPB.PrevHash = ab.PrevHash.Bytes()
	}

	if ab.FromHash != nil {
		accountBlockNetPB.FromHash = ab.FromHash.Bytes()
	}

	if ab.Hash != nil {
		accountBlockNetPB.Hash = ab.Hash.Bytes()
	}
	if ab.TokenId != nil {
		accountBlockNetPB.TokenId = ab.TokenId.Bytes()
	}

	if ab.Amount != nil {
		accountBlockNetPB.Amount = ab.Amount.Bytes()
	}

	if ab.Balance != nil {
		accountBlockNetPB.Amount = ab.Balance.Bytes()
	}

	if ab.SnapshotTimestamp != nil {
		accountBlockNetPB.SnapshotTimestamp = ab.SnapshotTimestamp.Bytes()
	}

	if ab.Signature != nil {
		accountBlockNetPB.SnapshotTimestamp = ab.SnapshotTimestamp.Bytes()
	}

	if ab.FAmount != nil {
		accountBlockNetPB.FAmount = ab.FAmount.Bytes()
	}

	return accountBlockNetPB
}

func (ab *AccountBlock) SetByNetPB (accountBlockNetPB *vitepb.AccountBlockNet) (error) {
	if accountBlockNetPB.Meta != nil {
		ab.Meta = &AccountBlockMeta {}

		ab.Meta.AccountId = &big.Int{}
		ab.Meta.AccountId.SetBytes(accountBlockNetPB.Meta.AccountId)

		ab.Meta.Height = &big.Int{}
		ab.Meta.Height.SetBytes(accountBlockNetPB.Meta.Height)
	}

	if accountBlockNetPB.To != nil {
		to, err := types.BytesToAddress(accountBlockNetPB.To)
		if err != nil {
			return nil
		}
		ab.To = &to
	}

	if accountBlockNetPB.PrevHash != nil {
		prevHash, err := types.BytesToHash(accountBlockNetPB.PrevHash)
		if err != nil {
			return nil
		}
		ab.PrevHash = &prevHash
	}

	if accountBlockNetPB.FromHash != nil {
		fromHash, err := types.BytesToHash(accountBlockNetPB.FromHash)
		if err != nil {
			return nil
		}
		ab.FromHash = &fromHash
	}

	if accountBlockNetPB.Hash != nil {
		hash, err := types.BytesToHash(accountBlockNetPB.Hash)
		if err != nil {
			return nil
		}
		ab.Hash = &hash
	}

	if accountBlockNetPB.TokenId != nil {
		tokenId, err := types.BytesToTokenTypeId(accountBlockNetPB.TokenId)
		if err != nil {
			return nil
		}
		ab.TokenId = &tokenId
	}

	if accountBlockNetPB.Amount != nil {
		ab.Amount = &big.Int{}
		ab.Amount.SetBytes(accountBlockNetPB.Amount)
	}

	if accountBlockNetPB.Balance != nil {
		ab.Balance = &big.Int{}
		ab.Balance.SetBytes(accountBlockNetPB.Balance)
	}

	if accountBlockNetPB.Data != "" {
		ab.Data = accountBlockNetPB.Data
	}

	if accountBlockNetPB.SnapshotTimestamp != nil {
		snapshotTimestamp, err := types.BytesToHash(accountBlockNetPB.SnapshotTimestamp)
		if err != nil {
			return nil
		}
		ab.SnapshotTimestamp = &snapshotTimestamp
	}

	ab.Timestamp = accountBlockNetPB.Timestamp

	ab.Signature = accountBlockNetPB.Signature

	ab.Nounce = accountBlockNetPB.Nounce

	ab.Difficulty = accountBlockNetPB.Difficulty

	if accountBlockNetPB.FAmount != nil {
		ab.FAmount = &big.Int{}
		ab.FAmount.SetBytes(accountBlockNetPB.FAmount)
	}

	return nil
}


func (ab *AccountBlock) NetDeserialize (buf []byte) (error)  {
	accountBlockNetPB := &vitepb.AccountBlockNet{}
	if err := proto.Unmarshal(buf, accountBlockNetPB); err != nil {
		return err
	}

	ab.SetByNetPB(accountBlockNetPB)

	return nil
}


func (ab *AccountBlock) NetSerialize () ([]byte, error)  {
	return proto.Marshal(ab.GetNetPB())
}

func (ab *AccountBlock) DbSerialize () ([]byte, error) {
	accountBlockPB := &vitepb.AccountBlockDb{
		Timestamp: ab.Timestamp,
		Data: ab.Data,

		Signature: ab.Signature,

		Nounce: ab.Nounce,
		Difficulty: ab.Difficulty,
	}

	if ab.Hash != nil {
		accountBlockPB.Hash = ab.Hash.Bytes()
	}
	if ab.PrevHash != nil {
		accountBlockPB.Hash = ab.PrevHash.Bytes()
	}
	if ab.FromHash != nil {
		accountBlockPB.Hash = ab.FromHash.Bytes()
	}

	if ab.Amount != nil {
		accountBlockPB.Amount = ab.Amount.Bytes()
	}

	if ab.To != nil {
		accountBlockPB.To = ab.To.Bytes()
	}

	if ab.TokenId != nil {
		accountBlockPB.TokenId = ab.TokenId.Bytes()
	}

	if ab.SnapshotTimestamp != nil {
		accountBlockPB.SnapshotTimestamp = ab.SnapshotTimestamp.Bytes()
	}

	if ab.Balance != nil {
		accountBlockPB.Balance = ab.Balance.Bytes()
	}

	if ab.FAmount != nil {
		accountBlockPB.FAmount = ab.FAmount.Bytes()
	}


	return proto.Marshal(accountBlockPB)
}



func (ab *AccountBlock) DbDeserialize (buf []byte) error {
	accountBlockPB := &vitepb.AccountBlockDb{}
	if err := proto.Unmarshal(buf, accountBlockPB); err != nil {
		return err
	}

	if accountBlockPB.To != nil {
		toAddress, err := types.BytesToAddress(accountBlockPB.To)
		if err != nil {
			return err
		}

		ab.To = &toAddress
	}


	if accountBlockPB.Hash != nil {
		hash, err := types.BytesToHash(accountBlockPB.Hash)
		if err != nil {
			return err
		}
		ab.Hash = &hash
	}

	if accountBlockPB.PrevHash != nil {
		prevHash, err := types.BytesToHash(accountBlockPB.PrevHash)
		if err != nil {
			return err
		}
		ab.PrevHash = &prevHash
	}

	if accountBlockPB.FromHash != nil {
		fromHash, err := types.BytesToHash(accountBlockPB.FromHash)
		if err != nil {
			return err
		}
		ab.FromHash = &fromHash
	}

	if accountBlockPB.TokenId != nil {
		tokenId, err := types.BytesToTokenTypeId(accountBlockPB.TokenId)
		if err != nil {
			return err
		}

		ab.TokenId = &tokenId
	}

	if accountBlockPB.Amount != nil {
		ab.Amount = &big.Int{}
		ab.Amount.SetBytes(accountBlockPB.Amount)
	}


	ab.Timestamp =  accountBlockPB.Timestamp

	if accountBlockPB.Balance != nil {
		ab.Balance = &big.Int{}
		ab.Balance.SetBytes(accountBlockPB.Balance)
	}


	ab.Data = accountBlockPB.Data

	if accountBlockPB.SnapshotTimestamp != nil {
		snapshotTimestamp, err := types.BytesToHash(accountBlockPB.SnapshotTimestamp)
		if err != nil {
			return err
		}
		ab.SnapshotTimestamp = &snapshotTimestamp
	}

	ab.Signature = accountBlockPB.Signature

	ab.Nounce = accountBlockPB.Nounce

	ab.Difficulty = accountBlockPB.Difficulty

	ab.FAmount = &big.Int{}
	ab.FAmount.SetBytes(accountBlockPB.FAmount)

	return nil
}


func GetGenesisBlockFirst () (*AccountBlock){
	return &AccountBlock{
		AccountAddress: GenesisSnapshotBlock.Producer,
		To: 			&MintageAddress,

		SnapshotTimestamp: GenesisSnapshotBlock.Hash,
		Timestamp: uint64(1532084800),
		Data: "{" +
			"\"tokenName\": \"vite\"," +
			"\"tokenSymbol\": \"VITE\"," +
			"\"owner\":\""+ GenesisSnapshotBlock.Producer.String() +"\"," +
			"\"decimals\": 18," +
			"\"tokenId\":\"" + MockViteTokenId.String() + "\"," +
			"\"totalSupply\": \"1000000000\"" +
			"}",
	}
}
func GetGenesisBlockSecond (fromHash *types.Hash) (*AccountBlock){
	return &AccountBlock{
		AccountAddress: GenesisSnapshotBlock.Producer,
		FromHash: fromHash,

		Timestamp: uint64(1532084900),
		SnapshotTimestamp: GenesisSnapshotBlock.Hash,
	}
}
