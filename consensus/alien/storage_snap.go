package alien

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/common/hexutil"
	"github.com/UltronGlow/UltronGlow-Origin/consensus"
	"github.com/UltronGlow/UltronGlow-Origin/core/state"
	"github.com/UltronGlow/UltronGlow-Origin/core/types"
	"github.com/UltronGlow/UltronGlow-Origin/ethdb"
	"github.com/UltronGlow/UltronGlow-Origin/log"
	"github.com/UltronGlow/UltronGlow-Origin/rlp"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/sha3"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

const (
	utgStorageDeclare        = "stReq"
	utgStorageExit           = "stExit"
	utgRentRequest           = "stRent"
	utgRentPg                = "stRentPg"
	utgRentReNew             = "stReNew"
	utgRentReNewPg           = "stReNewPg"
	utgRentRescind           = "stRescind"
	utgStorageRecoverValid   = "stReValid"
	utgStorageProof          = "stProof"
	utgStoragePrice          = "chPrice"
	storagePledgeRewardkey   = "storagePledgeReward-%d"
	storageLeaseRewardkey    = "storageLeaseReward-%d"
	revertSpaceLockRewardkey = "revertSpaceLockReward-%d"
	storageRatioskey         = "storageRatios-%d"
	revertExchangeSRTkey     = "revertExchangeSRT-%d"

	SPledgeNormal    = 0
	SPledgeExit      = 1
	SPledgeRemoving  = 5 //30-day verification failed
	SPledgeRetrun    = 6 //SRT and pledge deposit have been returned
	LeaseNotPledged  = 0
	LeaseNormal      = 1
	LeaseUserRescind = 2
	LeaseExpiration  = 3
	LeaseBreach      = 4
	LeaseReturn      = 6
)

var (
	totalSpaceProfitReward = new(big.Int).Mul(big.NewInt(1e+18), big.NewInt(10500000))
	gbTob                  = big.NewInt(1073741824)
	tb1b                   = big.NewInt(1099511627776)
	minPledgeStorageCapacity= decimal.NewFromInt(1099511627776)
	maxPledgeStorageCapacity= decimal.NewFromInt(1099511627776).Mul(decimal.NewFromInt(80))
	proofTimeOut = big.NewInt(1800)  //second
	storageBlockSize = "20"
)

type StorageData struct {
	StoragePledge map[common.Address]*SPledge `json:"spledge"`
	Hash          common.Hash                 `json:"validhash"`
}

/**
Storage pledge struct
*/
type SPledge struct {
	Address                     common.Address         `json:"address"`
	StorageSpaces               *SPledgeSpaces         `json:"storagespaces"`
	Number                      *big.Int               `json:"number"`
	TotalCapacity               *big.Int               `json:"totalcapacity"`
	Bandwidth                   *big.Int               `json:"bandwidth"`
	Price                       *big.Int               `json:"price"`
	StorageSize                 *big.Int               `json:"storagesize"`
	SpaceDeposit                *big.Int               `json:"spacedeposit"`
	Lease                       map[common.Hash]*Lease `json:"lease"`
	LastVerificationTime        *big.Int               `json:"lastverificationtime"`
	LastVerificationSuccessTime *big.Int               `json:"lastverificationsuccesstime"`
	ValidationFailureTotalTime  *big.Int               `json:"validationfailuretotaltime"`
	PledgeStatus                *big.Int               `json:"pledgeStatus"`
	Hash                        common.Hash            `json:"validhash"`
}

/**
 * Storage  space
 */
type SPledgeSpaces struct {
	Address                     common.Address               `json:"address"`
	StorageCapacity             *big.Int                     `json:"storagecapacity"`
	RootHash                    common.Hash                  `json:"roothash"`
	StorageFile                 map[common.Hash]*StorageFile `json:"storagefile"`
	LastVerificationTime        *big.Int                     `json:"lastverificationtime"`
	LastVerificationSuccessTime *big.Int                     `json:"lastverificationsuccesstime"`
	ValidationFailureTotalTime  *big.Int                     `json:"validationfailuretotaltime"`
	Hash                        common.Hash                  `json:"validhash"`
}

/**
 *Rental structure
 */
type Lease struct {
	Address                     common.Address               `json:"address"`
	DepositAddress              common.Address               `json:"depositaddress"`
	Capacity                    *big.Int                     `json:"capacity"`
	RootHash                    common.Hash                  `json:"roothash"`
	Deposit                     *big.Int                     `json:"deposit"`
	UnitPrice                   *big.Int                     `json:"unitprice"`
	Cost                        *big.Int                     `json:"cost"`
	Duration                    *big.Int                     `json:"duration"`
	StorageFile                 map[common.Hash]*StorageFile `json:"storagefile"`
	LeaseList                   map[common.Hash]*LeaseDetail `json:"leaselist"`
	LastVerificationTime        *big.Int                     `json:"lastverificationtime"`
	LastVerificationSuccessTime *big.Int                     `json:"lastverificationsuccesstime"`
	ValidationFailureTotalTime  *big.Int                     `json:"validationfailuretotaltime"`
	Status                      int                          `json:"status"`
	Hash                        common.Hash                  `json:"validhash"`
}

/**
 * Rental structure
 */
type StorageFile struct {
	Capacity                    *big.Int    `json:"capacity"`
	CreateTime                  *big.Int    `json:"createtime"`
	LastVerificationTime        *big.Int    `json:"lastverificationtime"`
	LastVerificationSuccessTime *big.Int    `json:"lastverificationsuccesstime"`
	ValidationFailureTotalTime  *big.Int    `json:"validationfailuretotaltime"`
	Hash                        common.Hash `json:"validhash"`
}

/**
 *  Lease list
 */
type LeaseDetail struct {
	RequestHash                common.Hash `json:"requesthash"`
	PledgeHash                 common.Hash `json:"pledgehash"`
	RequestTime                *big.Int    `json:"requesttime"`
	StartTime                  *big.Int    `json:"starttime"`
	Duration                   *big.Int    `json:"duration"`
	Cost                       *big.Int    `json:"cost"`
	Deposit                    *big.Int    `json:"deposit"`
	ValidationFailureTotalTime *big.Int    `json:"validationfailuretotaltime"`
	Revert                     int         `json:"revert"`
	Hash                       common.Hash `json:"validhash"`
}
type SPledgeRecord struct {
	PledgeAddr      common.Address `json:"pledgeAddr"`
	Address         common.Address `json:"address"`
	Price           *big.Int       `json:"price"`
	SpaceDeposit    *big.Int       `json:"spacedeposit"`
	StorageCapacity *big.Int       `json:"storagecapacity"`
	StorageSize     *big.Int       `json:"storagesize"`
	RootHash        common.Hash    `json:"roothash"`
	PledgeNumber    *big.Int       `json:"pledgeNumber"`
	Bandwidth       *big.Int       `json:"bandwidth"`
}
type SPledgeExitRecord struct {
	Address      common.Address `json:"address"`
	PledgeStatus *big.Int       `json:"pledgeStatus"`
}

type LeaseRequestRecord struct {
	Tenant   common.Address `json:"tenant"`
	Address  common.Address `json:"address"`
	Capacity *big.Int       `json:"capacity"`
	Duration *big.Int       `json:"duration"`
	Price    *big.Int       `json:"price"`
	Hash     common.Hash    `json:"hash"`
}

type LeasePledgeRecord struct {
	Address        common.Address `json:"address"`
	DepositAddress common.Address `json:"depositaddress"`
	Hash           common.Hash    `json:"hash"`
	Capacity       *big.Int       `json:"capacity"`
	RootHash       common.Hash    `json:"roothash"`
	BurnSRTAmount  *big.Int       `json:"burnsrtamount"`
	BurnAmount     *big.Int       `json:"burnamount"`
	Duration       *big.Int       `json:"duration"`
	BurnSRTAddress common.Address `json:"burnsrtaddress"`
	PledgeHash     common.Hash    `json:"pledgehash"`
	LeftCapacity   *big.Int       `json:"leftcapacity"`
	LeftRootHash   common.Hash    `json:"leftroothash"`
}
type LeaseRenewalPledgeRecord struct {
	Address        common.Address `json:"address"`
	Hash           common.Hash    `json:"hash"`
	Capacity       *big.Int       `json:"capacity"`
	RootHash       common.Hash    `json:"roothash"`
	BurnSRTAmount  *big.Int       `json:"burnsrtamount"`
	BurnAmount     *big.Int       `json:"burnamount"`
	Duration       *big.Int       `json:"duration"`
	BurnSRTAddress common.Address `json:"burnsrtaddress"`
	PledgeHash     common.Hash    `json:"pledgehash"`
}

type LeaseRenewalRecord struct {
	Address  common.Address `json:"address"`
	Duration *big.Int       `json:"duration"`
	Hash     common.Hash    `json:"hash"`
	Price    *big.Int       `json:"price"`
	Tenant   common.Address `json:"tenant"`
	NewHash  common.Hash    `json:"newhash"`
	Capacity *big.Int       `json:"capacity"`
}

type LeaseRescindRecord struct {
	Address common.Address `json:"address"`
	Hash    common.Hash    `json:"hash"`
}
type SExpireRecord struct {
	Address common.Address `json:"address"`
	Hash    common.Hash    `json:"hash"`
}
type SPledgeRecoveryRecord struct {
	Address       common.Address `json:"address"`
	LeaseHash     []common.Hash  `json:"leaseHash"`
	SpaceCapacity *big.Int       `json:"spaceCapacity"`
	RootHash      common.Hash    `json:"rootHash"`
	ValidNumber   *big.Int       `json:"validNumber"`
}
type StorageProofRecord struct {
	Address                     common.Address `json:"address"`
	LeaseHash                   common.Hash    `json:"leaseHash"`
	RootHash                    common.Hash    `json:"rootHash"`
	LastVerificationTime        *big.Int       `json:"lastverificationtime"`
	LastVerificationSuccessTime *big.Int       `json:"lastverificationsuccesstime"`
}
type StorageExchangePriceRecord struct {
	Address common.Address `json:"address"`
	Price   *big.Int       `json:"price"`
}

type StorageRatio struct {
	Capacity *big.Int        `json:"capacity"`
	Ratio    decimal.Decimal `json:"ratio"`
}

type SpaceRewardRecord struct {
	Target  common.Address `json:"target"`
	Amount  *big.Int       `json:"amount"`
	Revenue common.Address `json:"revenue"`
}

func (a *Alien) processStorageCustomTx(txDataInfo []string, headerExtra HeaderExtra, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, snapCache *Snapshot, number *big.Int, state *state.StateDB, chain consensus.ChainHeaderReader) HeaderExtra {
	if txDataInfo[posCategory] == utgRentRequest {
		headerExtra.LeaseRequest = a.processRentRequest(headerExtra.LeaseRequest, txDataInfo, txSender, tx, receipts, snapCache, number.Uint64())
	} else if txDataInfo[posCategory] == utgSRTExch {
		headerExtra.ExchangeSRT = a.processExchangeSRT(headerExtra.ExchangeSRT, txDataInfo, txSender, tx, receipts, state, snapCache)
	} else if txDataInfo[posCategory] == utgStorageDeclare {
		headerExtra.StoragePledge = a.declareStoragePledge(headerExtra.StoragePledge, txDataInfo, txSender, tx, receipts, state, snapCache, number, chain)
	} else if txDataInfo[posCategory] == utgStorageExit {
		headerExtra.StoragePledgeExit, headerExtra.ExchangeSRT = a.storagePledgeExit(headerExtra.StoragePledgeExit, headerExtra.ExchangeSRT, txDataInfo, txSender, tx, receipts, state, snapCache, number)
	} else if txDataInfo[posCategory] == utgRentPg {
		headerExtra.LeasePledge = a.processLeasePledge(headerExtra.LeasePledge, txDataInfo, txSender, tx, receipts, state, snapCache, number.Uint64())
	} else if txDataInfo[posCategory] == utgRentReNew {
		headerExtra.LeaseRenewal = a.processLeaseRenewal(headerExtra.LeaseRenewal, txDataInfo, txSender, tx, receipts, state, snapCache, number.Uint64())
	} else if txDataInfo[posCategory] == utgRentReNewPg {
		headerExtra.LeaseRenewalPledge = a.processLeaseRenewalPledge(headerExtra.LeaseRenewalPledge, txDataInfo, txSender, tx, receipts, state, snapCache, number.Uint64())
	} else if txDataInfo[posCategory] == utgRentRescind {
		headerExtra.LeaseRescind, headerExtra.ExchangeSRT = a.processLeaseRescind(headerExtra.LeaseRescind, headerExtra.ExchangeSRT, txDataInfo, txSender, tx, receipts, state, snapCache, number.Uint64())
	} else if txDataInfo[posCategory] == utgStorageRecoverValid {
		headerExtra.StorageRecoveryData = a.storageRecoveryCertificate(headerExtra.StorageRecoveryData, txDataInfo, txSender, tx, receipts, state, snapCache, number, chain)
	} else if txDataInfo[posCategory] == utgStorageProof {
		headerExtra.StorageProofRecord = a.applyStorageProof(headerExtra.StorageProofRecord, txDataInfo, txSender, tx, receipts, state, snapCache, number, chain)
	} else if txDataInfo[posCategory] == utgStoragePrice {
		headerExtra.StorageExchangePrice = a.exchangeStoragePrice(headerExtra.StorageExchangePrice, txDataInfo, txSender, tx, receipts, state, snapCache, number)

	}
	return headerExtra
}
func (snap *Snapshot) storageApply(headerExtra HeaderExtra, header *types.Header, db ethdb.Database) (*Snapshot, error) {
	calsnap, err := snap.calStorageVerificationCheck(headerExtra.StorageDataRoot, header.Number.Uint64(), snap.getBlockPreDay())
	if err != nil {
		log.Error("calStorageVerificationCheck", "err", err)
		return calsnap, err
	}
	snap.updateExchangeSRT(headerExtra.ExchangeSRT, header.Number, db)
	snap.updateStorageData(headerExtra.StoragePledge, db)
	snap.updateStoragePledgeExit(headerExtra.StoragePledgeExit, header.Number, db)
	snap.updateLeaseRequest(headerExtra.LeaseRequest, header.Number, db)
	snap.updateLeasePledge(headerExtra.LeasePledge, header.Number, db)
	snap.updateLeaseRenewal(headerExtra.LeaseRenewal, header.Number, db)
	snap.updateLeaseRenewalPledge(headerExtra.LeaseRenewalPledge, header.Number, db)
	snap.updateLeaseRescind(headerExtra.LeaseRescind, header.Number, db)
	snap.updateStorageRecoveryData(headerExtra.StorageRecoveryData, header.Number, db)
	snap.updateStorageProof(headerExtra.StorageProofRecord, header.Number, db)
	snap.updateStoragePrice(headerExtra.StorageExchangePrice, header.Number, db)
	return snap, nil
}
func (s *StorageData) checkSRent(sRent []LeaseRequestRecord, rent LeaseRequestRecord) bool {
	if _, ok := s.StoragePledge[rent.Address]; !ok {
		return false
	}
	//check price
	price := s.StoragePledge[rent.Address].Price
	if rent.Price.Cmp(price) < 0 {
		return false
	}
	if s.StoragePledge[rent.Address].PledgeStatus.Cmp(big.NewInt(SPledgeNormal))!=0{
		return false
	}
	//check Capacity
	rentCapacity := new(big.Int).Set(rent.Capacity)
	for _, item := range sRent {
		if item.Address == rent.Address {
			rentCapacity = new(big.Int).Add(rentCapacity, item.Capacity)
		}
	}
	storageSpaces := s.StoragePledge[rent.Address].StorageSpaces
	if storageSpaces.StorageCapacity.Cmp(rentCapacity) < 0 {
		return false
	}
	return true
}

func (s *StorageData) updateLeaseRequest(sRent []LeaseRequestRecord, number *big.Int, db ethdb.Database) {
	for _, item := range sRent {
		spledge, _ := s.StoragePledge[item.Address]
		if _, ok := spledge.Lease[item.Hash]; !ok {
			zero := big.NewInt(0)
			leaseDetail := LeaseDetail{
				RequestHash:                item.Hash,
				PledgeHash:                 common.Hash{},
				RequestTime:                number,
				StartTime:                  big.NewInt(0),
				Duration:                   item.Duration,
				Cost:                       zero,
				Deposit:                    zero,
				ValidationFailureTotalTime: big.NewInt(0),
			}
			LeaseList := make(map[common.Hash]*LeaseDetail)
			LeaseList[item.Hash] = &leaseDetail
			spledge.Lease[item.Hash] = &Lease{
				Address:                     item.Tenant,
				Capacity:                    item.Capacity,
				RootHash:                    common.Hash{},
				Deposit:                     zero,
				UnitPrice:                   item.Price,
				Cost:                        zero,
				Duration:                    zero,
				StorageFile:                 make(map[common.Hash]*StorageFile),
				LeaseList:                   LeaseList,
				LastVerificationTime:        zero,
				LastVerificationSuccessTime: zero,
				ValidationFailureTotalTime:  zero,
				Status:                      LeaseNotPledged,
			}
			s.accumulateLeaseDetailHash(item.Address, item.Hash, LeaseList[item.Hash])
		}
	}
	s.accumulateHeaderHash()
}
func (s *StorageData) checkSRentPg(currentSRentPg []LeasePledgeRecord, sRentPg LeasePledgeRecord, txSender common.Address, revenueStorage map[common.Address]*RevenueParameter, exchRate uint32) (*big.Int, *big.Int, *big.Int, common.Address, bool) {
	nilHash := common.Address{}
	for _, item := range currentSRentPg {
		if item.Address == sRentPg.Address {
			return nil, nil, nil, nilHash, false
		}
	}
	//checkCapacity
	if _, ok := s.StoragePledge[sRentPg.Address]; !ok {
		return nil, nil, nil, nilHash, false
	}
	if _, ok := s.StoragePledge[sRentPg.Address].Lease[sRentPg.Hash]; !ok {
		return nil, nil, nil, nilHash, false
	}
	lease := s.StoragePledge[sRentPg.Address].Lease[sRentPg.Hash]
	if lease.Capacity.Cmp(sRentPg.Capacity) != 0 {
		return nil, nil, nil, nilHash, false
	}
	storageCapacity := s.StoragePledge[sRentPg.Address].StorageSpaces.StorageCapacity
	leftCapacity := new(big.Int).Sub(storageCapacity, sRentPg.Capacity)
	if leftCapacity.Cmp(sRentPg.LeftCapacity) != 0 {
		return nil, nil, nil, nilHash, false
	}
	if lease.Deposit.Cmp(big.NewInt(0)) > 0 {
		return nil, nil, nil, nilHash, false
	}
	//checkowner
	sRentPg.DepositAddress = txSender
	//checkfileproof  todo

	//Calculate the pledge deposit
	leaseDetail := lease.LeaseList[sRentPg.Hash]
	srtAmount := new(big.Int).Mul(leaseDetail.Duration, lease.UnitPrice)
	srtAmount = new(big.Int).Mul(srtAmount, lease.Capacity)
	srtAmount = new(big.Int).Div(srtAmount, gbTob)
	amount := new(big.Int).Div(new(big.Int).Mul(srtAmount, big.NewInt(10000)), big.NewInt(int64(exchRate)))
	return srtAmount, amount, leaseDetail.Duration, lease.Address, true
}

func (s *StorageData) updateLeasePledge(pg []LeasePledgeRecord, number *big.Int, db ethdb.Database) {
	for _, sRentPg := range pg {
		if _, ok := s.StoragePledge[sRentPg.Address]; !ok {
			continue
		}
		if _, ok := s.StoragePledge[sRentPg.Address].Lease[sRentPg.Hash]; !ok {
			continue
		}
		lease := s.StoragePledge[sRentPg.Address].Lease[sRentPg.Hash]
		lease.RootHash = sRentPg.RootHash
		lease.Deposit = new(big.Int).Add(lease.Deposit, sRentPg.BurnAmount)
		lease.Cost = new(big.Int).Add(lease.Cost, sRentPg.BurnSRTAmount)
		lease.Duration = new(big.Int).Add(lease.Duration, sRentPg.Duration)
		if _, ok := lease.StorageFile[sRentPg.RootHash]; !ok {
			lease.StorageFile[sRentPg.RootHash] = &StorageFile{
				Capacity:                    lease.Capacity,
				CreateTime:                  number,
				LastVerificationTime:        number,
				LastVerificationSuccessTime: number,
				ValidationFailureTotalTime:  big.NewInt(0),
			}
			s.accumulateLeaseStorageFileHash(sRentPg.Address, sRentPg.Hash, lease.StorageFile[sRentPg.RootHash])
		}
		leaseDetail := lease.LeaseList[sRentPg.Hash]
		leaseDetail.Cost = new(big.Int).Add(leaseDetail.Cost, sRentPg.BurnSRTAmount)
		leaseDetail.Deposit = new(big.Int).Add(leaseDetail.Deposit, sRentPg.BurnAmount)
		leaseDetail.PledgeHash = sRentPg.PledgeHash
		leaseDetail.StartTime = number
		lease.LastVerificationTime = number
		lease.LastVerificationSuccessTime = number
		lease.DepositAddress = sRentPg.DepositAddress
		lease.Status = LeaseNormal
		s.accumulateLeaseDetailHash(sRentPg.Address, sRentPg.Hash, leaseDetail)
		storageSpaces := s.StoragePledge[sRentPg.Address].StorageSpaces
		storageSpaces.StorageCapacity = sRentPg.LeftCapacity
		storageSpaces.RootHash = sRentPg.LeftRootHash
		storageSpaces.StorageFile = make(map[common.Hash]*StorageFile, 1)
		storageSpaces.StorageFile[sRentPg.LeftRootHash] = &StorageFile{
			Capacity:                    sRentPg.LeftCapacity,
			CreateTime:                  number,
			LastVerificationTime:        number,
			LastVerificationSuccessTime: number,
			ValidationFailureTotalTime:  big.NewInt(0),
		}
		s.accumulateSpaceStorageFileHash(sRentPg.Address, storageSpaces.StorageFile[sRentPg.LeftRootHash])
	}
	s.accumulateHeaderHash()
}
func (s *StorageData) checkSRentReNew(currentSRentReNew []LeaseRenewalRecord, sRentReNew LeaseRenewalRecord, txSender common.Address, number uint64, blockPerday uint64) (common.Address, bool) {
	nilHash := common.Address{}
	if _, ok := s.StoragePledge[sRentReNew.Address]; !ok {
		return nilHash, false
	}
	if s.StoragePledge[sRentReNew.Address].PledgeStatus.Cmp(big.NewInt(SPledgeNormal))!=0{
		return nilHash,false
	}
	if _, ok := s.StoragePledge[sRentReNew.Address].Lease[sRentReNew.Hash]; !ok {
		return nilHash, false
	}
	lease := s.StoragePledge[sRentReNew.Address].Lease[sRentReNew.Hash]
	if lease.Address != txSender {
		return nilHash, false
	}
	if lease.Status == LeaseNotPledged || lease.Status == LeaseUserRescind || lease.Status == LeaseExpiration || lease.Status == LeaseReturn {
		return nilHash, false
	}
	for _, rentnew := range currentSRentReNew {
		if rentnew.Hash == sRentReNew.Hash {
			return nilHash, false
		}
	}
	for _, detail := range lease.LeaseList {
		if detail.Deposit.Cmp(big.NewInt(0)) <= 0 {
			return nilHash, false
		}
	}
	startTime := big.NewInt(0)
	duration := big.NewInt(0)
	for _, leaseDetail := range lease.LeaseList {
		if leaseDetail.Deposit.Cmp(big.NewInt(0)) > 0 && leaseDetail.StartTime.Cmp(startTime) > 0 {
			startTime = leaseDetail.StartTime
			duration = new(big.Int).Mul(leaseDetail.Duration, new(big.Int).SetUint64(blockPerday))
		}
	}
	if startTime.Cmp(big.NewInt(0)) == 0 {
		return nilHash, false
	}
	duration90 := new(big.Int).Mul(duration, big.NewInt(rentRenewalExpires))
	duration90 = new(big.Int).Div(duration90, big.NewInt(100))
	reNewNumber := new(big.Int).Add(startTime, duration90)
	if reNewNumber.Cmp(new(big.Int).SetUint64(number)) > 0 {
		return nilHash, false
	}

	return lease.Address, true
}

func (s *StorageData) updateLeaseRenewal(reNew []LeaseRenewalRecord, number *big.Int, db ethdb.Database, blockPerDay uint64) {
	for _, item := range reNew {
		spledge, _ := s.StoragePledge[item.Address]
		if lease, ok := spledge.Lease[item.Hash]; ok {
			zero := big.NewInt(0)
			leaseDetail := LeaseDetail{
				RequestHash:                item.NewHash,
				PledgeHash:                 common.Hash{},
				RequestTime:                number,
				StartTime:                  big.NewInt(0),
				Duration:                   item.Duration,
				Cost:                       zero,
				Deposit:                    zero,
				ValidationFailureTotalTime: zero,
			}
			LeaseList := lease.LeaseList
			LeaseList[item.NewHash] = &leaseDetail
			s.accumulateLeaseDetailHash(item.Address, item.Hash, LeaseList[item.NewHash])
		}
	}
	s.accumulateHeaderHash()
}
func NewStorageSnap() *StorageData {
	return &StorageData{
		StoragePledge: make(map[common.Address]*SPledge),
	}
}
func (s *StorageData) copy() *StorageData {
	clone := &StorageData{
		StoragePledge: make(map[common.Address]*SPledge),
		Hash:          s.Hash,
	}
	for address, spledge := range s.StoragePledge {
		clone.StoragePledge[address] = &SPledge{
			Address: spledge.Address,
			StorageSpaces: &SPledgeSpaces{
				Address:                     spledge.StorageSpaces.Address,
				StorageCapacity:             spledge.StorageSpaces.StorageCapacity,
				RootHash:                    spledge.StorageSpaces.RootHash,
				StorageFile:                 make(map[common.Hash]*StorageFile),
				LastVerificationTime:        spledge.StorageSpaces.LastVerificationTime,
				LastVerificationSuccessTime: spledge.StorageSpaces.LastVerificationSuccessTime,
				ValidationFailureTotalTime:  spledge.StorageSpaces.ValidationFailureTotalTime,
				Hash:                        spledge.StorageSpaces.Hash,
			},
			Number:                      spledge.Number,
			TotalCapacity:               spledge.TotalCapacity,
			Bandwidth:                   spledge.Bandwidth,
			Price:                       spledge.Price,
			StorageSize:                 spledge.StorageSize,
			SpaceDeposit:                spledge.SpaceDeposit,
			Lease:                       make(map[common.Hash]*Lease),
			LastVerificationTime:        spledge.LastVerificationTime,
			LastVerificationSuccessTime: spledge.LastVerificationSuccessTime,
			ValidationFailureTotalTime:  spledge.ValidationFailureTotalTime,
			PledgeStatus:                spledge.PledgeStatus,
			Hash:                        spledge.Hash,
		}

		storageFiles := s.StoragePledge[address].StorageSpaces.StorageFile
		for hash, storageFile := range storageFiles {
			if _, ok := clone.StoragePledge[address].StorageSpaces.StorageFile[hash]; !ok {
				clone.StoragePledge[address].StorageSpaces.StorageFile[hash] = &StorageFile{
					Capacity:                    storageFile.Capacity,
					CreateTime:                  storageFile.CreateTime,
					LastVerificationTime:        storageFile.LastVerificationTime,
					LastVerificationSuccessTime: storageFile.LastVerificationSuccessTime,
					ValidationFailureTotalTime:  storageFile.ValidationFailureTotalTime,
					Hash:                        storageFile.Hash,
				}
			}
		}
		leases := s.StoragePledge[address].Lease
		for hash, lease := range leases {
			if _, ok := clone.StoragePledge[address].Lease[hash]; !ok {
				clone.StoragePledge[address].Lease[hash] = &Lease{
					Address:                     lease.Address,
					DepositAddress:              lease.DepositAddress,
					Capacity:                    lease.Capacity,
					RootHash:                    lease.RootHash,
					Deposit:                     lease.Deposit,
					UnitPrice:                   lease.UnitPrice,
					Cost:                        lease.Cost,
					Duration:                    lease.Duration,
					StorageFile:                 make(map[common.Hash]*StorageFile),
					LeaseList:                   make(map[common.Hash]*LeaseDetail),
					LastVerificationTime:        lease.LastVerificationTime,
					LastVerificationSuccessTime: lease.LastVerificationSuccessTime,
					ValidationFailureTotalTime:  lease.ValidationFailureTotalTime,
					Status:                      lease.Status,
					Hash:                        lease.Hash,
				}

				storageFiles2 := lease.StorageFile
				cloneSF := clone.StoragePledge[address].Lease[hash]
				for hash2, storageFile2 := range storageFiles2 {
					if _, ok2 := cloneSF.StorageFile[hash2]; !ok2 {
						cloneSF.StorageFile[hash2] = &StorageFile{
							Capacity:                    storageFile2.Capacity,
							CreateTime:                  storageFile2.CreateTime,
							LastVerificationTime:        storageFile2.LastVerificationTime,
							LastVerificationSuccessTime: storageFile2.LastVerificationSuccessTime,
							ValidationFailureTotalTime:  storageFile2.ValidationFailureTotalTime,
							Hash:                        storageFile2.Hash,
						}
					}
				}

				leaseLists := lease.LeaseList
				cloneLease := clone.StoragePledge[address].Lease[hash]
				for hash3, leaseDetail3 := range leaseLists {
					if _, ok2 := cloneLease.LeaseList[hash3]; !ok2 {
						cloneLease.LeaseList[hash3] = &LeaseDetail{
							RequestHash:                leaseDetail3.RequestHash,
							PledgeHash:                 leaseDetail3.PledgeHash,
							RequestTime:                leaseDetail3.RequestTime,
							StartTime:                  leaseDetail3.StartTime,
							Duration:                   leaseDetail3.Duration,
							Cost:                       leaseDetail3.Cost,
							Deposit:                    leaseDetail3.Deposit,
							ValidationFailureTotalTime: leaseDetail3.ValidationFailureTotalTime,
							Revert:                     leaseDetail3.Revert,
							Hash:                       leaseDetail3.Hash,
						}
					}
				}
			}
		}
	}
	return clone
}

func (a *Alien) declareStoragePledge(currStoragePledge []SPledgeRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot, blocknumber *big.Int, chain consensus.ChainHeaderReader) []SPledgeRecord {
	if len(txDataInfo) < 11 {
		log.Warn("declareStoragePledge", "parameter error len=", len(txDataInfo))
		return currStoragePledge
	}
	peledgeAddr := common.HexToAddress(txDataInfo[3])
	if _, ok := snap.StorageData.StoragePledge[peledgeAddr]; ok {
		log.Warn("Storage Pledge repeat", " peledgeAddr", peledgeAddr)
		return currStoragePledge
	}
	var bigPrice *big.Int
	if price, err := decimal.NewFromString(txDataInfo[4]); err != nil {
		log.Warn("Storage Pledge price wrong", "price", txDataInfo[4])
		return currStoragePledge
	} else {
		bigPrice = price.BigInt()
	}
	if bigPrice.Cmp(snap.SystemConfig.Deposit[sscEnumStoragePrice]) < 0 || bigPrice.Cmp(new(big.Int).Mul(snap.SystemConfig.Deposit[sscEnumStoragePrice], big.NewInt(10))) > 0 {
		log.Warn("price is set too high", " price", bigPrice)
		return currStoragePledge
	}
	storageCapacity, err := decimal.NewFromString(txDataInfo[5])
	if err != nil {
		log.Warn("Storage Pledge storageCapacity format error", "storageCapacity", txDataInfo[5])
		return currStoragePledge
	}

	if storageCapacity.Cmp(minPledgeStorageCapacity)<0 ||storageCapacity.Cmp(maxPledgeStorageCapacity)>0{
		log.Warn("Storage Pledge storageCapacity error", "storageCapacity",storageCapacity,"minPledgeStorageCapacity",minPledgeStorageCapacity,"maxPledgeStorageCapacity",maxPledgeStorageCapacity)
		return currStoragePledge
	}
	startPkNumber := txDataInfo[6]
	pkNonce,err:= decimal.NewFromString(txDataInfo[7])
	if err!=nil {
		log.Warn("Storage Pledge package nonce error", "pkNonce",txDataInfo[7])
		return currStoragePledge
	}
	pkBlockHash := txDataInfo[8]
	verifyData := txDataInfo[9]
	verifyDataArr := strings.Split(verifyData, ",")
	pkHeader := chain.GetHeaderByHash(common.HexToHash(pkBlockHash))
	if pkHeader == nil {
		log.Warn("Storage Pledge", "pkBlockHash is not exist", pkBlockHash)
		return currStoragePledge
	}
	if verifyDataArr[4]!= storageBlockSize {
		log.Warn("Storage Pledge storageBlockSize error", "storageBlockSize", storageBlockSize,"verifyDataArr[4]",verifyDataArr[4])
		return currStoragePledge
	}
	if pkHeader.Number.String() != startPkNumber || pkHeader.Nonce.Uint64()!=pkNonce.BigInt().Uint64(){
		log.Warn("Storage Pledge  packege param compare error", "startPkNumber", startPkNumber, "pkNonce", pkNonce, "pkBlockHash", pkBlockHash, " chain", pkHeader.Number)
		return currStoragePledge
	}
	rootHash := verifyDataArr[len(verifyDataArr)-1]
	if !verifyPocString(startPkNumber, txDataInfo[7], pkBlockHash, verifyData, rootHash, txDataInfo[3]) {
		log.Warn("Storage Pledge  verifyPoc Faild", "startPkNumber", startPkNumber, "pkNonce", pkNonce, "pkBlockHash", pkBlockHash)
		return currStoragePledge
	}
	storageSize, err := decimal.NewFromString(verifyDataArr[4])
	if err != nil {
		log.Warn("Storage Pledge storageSize format error", "storageSize", verifyDataArr[4])
		return currStoragePledge
	}
	bandwidth, err := decimal.NewFromString(txDataInfo[10])

	if err != nil || bandwidth.BigInt().Cmp(big.NewInt(0)) <= 0 {
		log.Warn("Storage Pledge  bandwidth error", "bandwidth", bandwidth)
		return currStoragePledge
	}
	totalStorage := big.NewInt(0)
	for _, spledge := range snap.StorageData.StoragePledge {
		totalStorage = new(big.Int).Add(totalStorage, spledge.TotalCapacity)
	}
	pledgeAmount := calStPledgeAmount(storageCapacity, snap, decimal.NewFromBigInt(totalStorage, 0), blocknumber)
	if state.GetBalance(txSender).Cmp(pledgeAmount) < 0 {
		log.Warn("Claimed sotrage", "balance", state.GetBalance(txSender))
		return currStoragePledge
	}
	state.SetBalance(txSender, new(big.Int).Sub(state.GetBalance(txSender), pledgeAmount))
	topics := make([]common.Hash, 3)
	topics[0].UnmarshalText([]byte("0x6d385a58ea1e7560a01c5a9d543911d47c1b86c5899c0b2df932dab4d7c2f323"))
	topics[1].SetBytes(peledgeAddr.Bytes())
	topics[2].SetBytes(pledgeAmount.Bytes())
	a.addCustomerTxLog(tx, receipts, topics, nil)
	storageRecord := SPledgeRecord{
		PledgeAddr:      txSender,
		Address:         peledgeAddr,
		Price:           bigPrice,
		SpaceDeposit:    pledgeAmount,
		StorageCapacity: storageCapacity.BigInt(),
		StorageSize:     storageSize.BigInt(),
		RootHash:        common.HexToHash(rootHash),
		PledgeNumber:    blocknumber,
		Bandwidth:       bandwidth.BigInt(),
	}
	currStoragePledge = append(currStoragePledge, storageRecord)
	return currStoragePledge
}
func (s *Snapshot) updateStorageData(pledgeRecord []SPledgeRecord, db ethdb.Database) {
	if pledgeRecord == nil || len(pledgeRecord) == 0 {
		return
	}
	for _, record := range pledgeRecord {
		storageFile := make(map[common.Hash]*StorageFile)
		storageFile[record.RootHash] = &StorageFile{
			Capacity:                    record.StorageCapacity,
			CreateTime:                  record.PledgeNumber,
			LastVerificationTime:        record.PledgeNumber,
			LastVerificationSuccessTime: record.PledgeNumber,
			ValidationFailureTotalTime:  big.NewInt(0),
		}

		space := &SPledgeSpaces{
			Address:                     record.Address,
			StorageCapacity:             record.StorageCapacity,
			RootHash:                    record.RootHash,
			StorageFile:                 storageFile,
			LastVerificationTime:        record.PledgeNumber,
			LastVerificationSuccessTime: record.PledgeNumber,
			ValidationFailureTotalTime:  big.NewInt(0),
		}
		storagepledge := &SPledge{
			Address:                     record.PledgeAddr,
			StorageSpaces:               space,
			Number:                      record.PledgeNumber,
			TotalCapacity:               record.StorageCapacity,
			Price:                       record.Price,
			StorageSize:                 record.StorageSize,
			SpaceDeposit:                record.SpaceDeposit,
			Lease:                       make(map[common.Hash]*Lease),
			LastVerificationTime:        record.PledgeNumber,
			LastVerificationSuccessTime: record.PledgeNumber,
			ValidationFailureTotalTime:  big.NewInt(0),
			PledgeStatus:                big.NewInt(SPledgeNormal),
			Bandwidth:                   record.Bandwidth,
		}
		s.StorageData.StoragePledge[record.Address] = storagepledge
		s.StorageData.accumulateSpaceStorageFileHash(record.Address, storageFile[record.RootHash]) //update file -->  space -- pledge
		log.Info("storage pledge save successfully!", "s.StorageData.StoragePledge", s.StorageData.StoragePledge)
	}
	s.StorageData.accumulateHeaderHash() //update all  to header valid root
}

func calStPledgeAmount(totalCapacity decimal.Decimal, snap *Snapshot, total decimal.Decimal, blockNumPer *big.Int) *big.Int {
	scale := decimal.NewFromBigInt(snap.SystemConfig.Deposit[sscEnumPStoragePledgeID], 0).Div(decimal.NewFromInt(10)) //0.1
	blockNumPerYear := secondsPerYear / snap.config.Period
	//1.25 UTG
	defaultTbAmount := decimal.NewFromFloat(1250000000000000000)
	tbPledgeNum := defaultTbAmount //1TB  UTG
	if blockNumPer.Uint64() > blockNumPerYear {
		totalSpace := total.Div(decimal.NewFromInt(1099511627776)) // B-> TB
		if totalSpace.Cmp(decimal.NewFromInt(0))>0 {
			calTbPledgeNum := decimal.NewFromBigInt(snap.FlowHarvest, 0).Mul(scale).Div(totalSpace)
			if calTbPledgeNum.Cmp(defaultTbAmount) < 0 {
				tbPledgeNum = calTbPledgeNum
			}
		}
	}

	return (totalCapacity.Div(decimal.NewFromInt(1099511627776))).Mul(tbPledgeNum).BigInt()
}

func (a *Alien) storagePledgeExit(storagePledgeExit []SPledgeExitRecord, exchangeSRT []ExchangeSRTRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot, blocknumber *big.Int) ([]SPledgeExitRecord, []ExchangeSRTRecord) {
	if len(txDataInfo) < 4 {
		log.Warn("storage Pledge exit", "parameter error", len(txDataInfo))
		return storagePledgeExit, exchangeSRT
	}
	pledgeAddr := common.HexToAddress(txDataInfo[3])
	if revenue, ok := snap.RevenueStorage[pledgeAddr]; ok {
		log.Warn("storage Pledge exit", "bind Revenue address", revenue.RevenueAddress)
		return storagePledgeExit, exchangeSRT
	}
	if pledgeAddr != txSender {
		log.Warn("storagePledgeExit  no role", " txSender", txSender)
		return storagePledgeExit, exchangeSRT
	}
	storagepledge := snap.StorageData.StoragePledge[pledgeAddr]
	if storagepledge== nil {
		log.Warn("storagePledgeExit  pledgeAddr not find  ", " pledgeAddr", pledgeAddr)
		return storagePledgeExit, exchangeSRT
	}
	if storagepledge.PledgeStatus.Cmp(big.NewInt(SPledgeExit)) == 0 {
		log.Warn("storagePledgeExit  has exit", " pledgeAddr", pledgeAddr)
		return storagePledgeExit, exchangeSRT
	}
	if storagepledge == nil {
		log.Warn("storagePledgeExit  not find pledge", " pledgeAddr", pledgeAddr)
		return storagePledgeExit, exchangeSRT
	}
	leaseStatus := false
	for _, lease := range storagepledge.Lease {
		if lease.Status != LeaseUserRescind && lease.Status != LeaseExpiration && lease.Status != LeaseReturn {
			leaseStatus = true
			break
		}
	}
	if leaseStatus {
		log.Warn("storagePledgeExit There are still open leases ", " pledgeAddr", pledgeAddr)
		return storagePledgeExit, exchangeSRT
	}
	storagePledgeExit = append(storagePledgeExit, SPledgeExitRecord{
		Address:      pledgeAddr,
		PledgeStatus: big.NewInt(1),
	})
	topics := make([]common.Hash, 3)
	topics[0].UnmarshalText([]byte("0Xff21066efa593b073738a132cf978c90bcbae2c98f6956e8a9e8663ade52f33c"))
	topics[1].SetBytes(pledgeAddr.Bytes())
	topics[2].SetBytes([]byte("0"))
	a.addCustomerTxLog(tx, receipts, topics, nil)
	return storagePledgeExit, exchangeSRT
}
func (s *Snapshot) updateStoragePledgeExit(storagePledgeExit []SPledgeExitRecord, headerNumber *big.Int, db ethdb.Database) {
	if storagePledgeExit == nil || len(storagePledgeExit) == 0 {
		return
	}
	for _, pledgeExit := range storagePledgeExit {
		s.StorageData.StoragePledge[pledgeExit.Address].PledgeStatus = pledgeExit.PledgeStatus
		s.StorageData.accumulatePledgeHash(pledgeExit.Address)
	}
	s.StorageData.accumulateHeaderHash()
}
func (a *Alien) processRentRequest(currentSRent []LeaseRequestRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, snap *Snapshot, number uint64) []LeaseRequestRecord {
	if len(txDataInfo) < 7 {
		log.Warn("sRent", "parameter number", len(txDataInfo))
		return currentSRent
	}
	sRent := LeaseRequestRecord{
		Tenant:   txSender,
		Address:  common.Address{},
		Capacity: big.NewInt(0),
		Duration: big.NewInt(0),
		Price:    big.NewInt(0),
		Hash:     tx.Hash(),
	}
	postion := 3
	if err := sRent.Address.UnmarshalText1([]byte(txDataInfo[postion])); err != nil {
		log.Warn("sRent", "address", txDataInfo[postion])
		return currentSRent
	}
	postion++
	if capacity, err := decimal.NewFromString(txDataInfo[postion]); err != nil {
		log.Warn("sRentPg", "Capacity", txDataInfo[postion])
		return currentSRent
	} else {
		sRent.Capacity = capacity.BigInt()
	}
	postion++
	if duration, err := strconv.ParseUint(txDataInfo[postion], 10, 64); err != nil {
		log.Warn("sRent", "duration", txDataInfo[postion])
		return currentSRent
	} else {
		sRent.Duration = new(big.Int).SetUint64(duration)
	}
	if sRent.Duration.Cmp(snap.SystemConfig.Deposit[sscEnumMinimumRent]) < 0 {
		log.Warn("sRent", "Duration", sRent.Duration)
		return currentSRent
	}
	postion++
	if price, err := decimal.NewFromString(txDataInfo[postion]); err != nil {
		log.Warn("sRent", "price", txDataInfo[postion])
		return currentSRent
	} else {
		sRent.Price = price.BigInt()
	}
	if sRent.Price.Cmp(new(big.Int).Mul(snap.SystemConfig.Deposit[sscEnumStoragePrice], big.NewInt(10))) > 0 {
		log.Warn("price is set too high", " price", sRent.Price)
		return currentSRent
	}
	//checkSRT
	if !snap.SRTIndex.checkEnoughSRT(currentSRent, sRent, number-1, a.db) {
		log.Warn("sRent", "checkEnoughSRT fail", sRent.Tenant)
		return currentSRent
	}
	//checkPledge
	if snap.StorageData.checkSRent(currentSRent, sRent) {
		topics := make([]common.Hash, 2)
		topics[0].UnmarshalText([]byte("0x24d91fe07adb5ec81f7c1724a69e7c307c289ff524f9ecb2519e631ba3f7f3d1"))
		topics[1].SetBytes(sRent.Address.Bytes())
		a.addCustomerTxLog(tx, receipts, topics, nil)
		currentSRent = append(currentSRent, sRent)
	} else {
		log.Warn("sRent", "checkSRent fail", sRent.Address)
	}
	return currentSRent
}
func (a *Alien) processExchangeSRT(currentExchangeSRT []ExchangeSRTRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot) []ExchangeSRTRecord {
	utgPosExchValue := 4
	if len(txDataInfo) <= utgPosExchValue {
		log.Warn("Exchange UTG to SRT fail", "parameter number", len(txDataInfo))
		return currentExchangeSRT
	}
	exchangeSRT := ExchangeSRTRecord{
		Target: common.Address{},
		Amount: big.NewInt(0),
	}
	if err := exchangeSRT.Target.UnmarshalText1([]byte(txDataInfo[3])); err != nil {
		log.Warn("Exchange UTG to SRT fail", "address", txDataInfo[3])
		return currentExchangeSRT
	}
	amount := big.NewInt(0)
	var err error
	if amount, err = hexutil.UnmarshalText1([]byte(txDataInfo[utgPosExchValue])); err != nil {
		log.Warn("Exchange UTG to SRT fail", "number", txDataInfo[utgPosExchValue])
		return currentExchangeSRT
	}
	if state.GetBalance(txSender).Cmp(amount) < 0 {
		log.Warn("Exchange UTG to SRT fail", "balance", state.GetBalance(txSender))
		return currentExchangeSRT
	}
	exchangeSRT.Amount = new(big.Int).Div(new(big.Int).Mul(amount, big.NewInt(int64(snap.SystemConfig.ExchRate))), big.NewInt(10000))
	state.SetBalance(txSender, new(big.Int).Sub(state.GetBalance(txSender), amount))
	topics := make([]common.Hash, 3)
	topics[0].UnmarshalText([]byte("0x1ebef91bab080007829976060bb3c203fd4d5b8395c552e10f5134e188428147")) //web3.sha3("ExchangeSRT(address,uint256)")
	topics[1].SetBytes(txSender.Bytes())
	topics[2].SetBytes(exchangeSRT.Target.Bytes())
	dataList := make([]common.Hash, 2)
	dataList[0].SetBytes(amount.Bytes())
	dataList[1].SetBytes(exchangeSRT.Amount.Bytes())
	data := dataList[0].Bytes()
	data = append(data, dataList[1].Bytes()...)
	a.addCustomerTxLog(tx, receipts, topics, data)
	currentExchangeSRT = append(currentExchangeSRT, exchangeSRT)
	return currentExchangeSRT
}

func (a *Alien) processLeasePledge(currentSRentPg []LeasePledgeRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot, number uint64) []LeasePledgeRecord {
	if len(txDataInfo) < 9 {
		log.Warn("sRentPg", "parameter number", len(txDataInfo))
		return currentSRentPg
	}
	sRentPg := LeasePledgeRecord{
		Address:        common.Address{},
		DepositAddress: txSender,
		Hash:           common.Hash{},
		Capacity:       big.NewInt(0),
		RootHash:       common.Hash{},
		BurnSRTAmount:  big.NewInt(0),
		Duration:       big.NewInt(0),
		BurnSRTAddress: common.Address{},
		PledgeHash:     tx.Hash(),
		LeftCapacity:   big.NewInt(0),
		LeftRootHash:   common.Hash{},
	}
	postion := 3
	if err := sRentPg.Address.UnmarshalText1([]byte(txDataInfo[postion])); err != nil {
		log.Warn("sRentPg", "Hash", txDataInfo[postion])
		return currentSRentPg
	}
	postion++
	sRentPg.Hash = common.HexToHash(txDataInfo[postion])
	postion++
	if capacity, err := decimal.NewFromString(txDataInfo[postion]); err != nil {
		log.Warn("sRentPg", "Capacity", txDataInfo[postion])
		return currentSRentPg
	} else {
		sRentPg.Capacity = capacity.BigInt()
	}
	postion++
	if rootHash, ok := snap.StorageData.verifyParamsStoragePoc(txDataInfo, postion, tx.Nonce()); !ok {
		log.Warn("sRentPg verify fail", " RootHash1", rootHash)
		return currentSRentPg
	} else {
		sRentPg.RootHash = rootHash
	}
	postion++
	if leftCapacity, err := decimal.NewFromString(txDataInfo[postion]); err != nil {
		log.Warn("sRentPg", "Capacity", txDataInfo[postion])
		return currentSRentPg
	} else {
		sRentPg.LeftCapacity = leftCapacity.BigInt()
	}
	postion++
	if rootHash, ok := snap.StorageData.verifyParamsStoragePoc(txDataInfo, postion, tx.Nonce()); !ok {
		log.Warn("sRentPg verify fail", " RootHash2", rootHash)
		return currentSRentPg
	} else {
		sRentPg.LeftRootHash = rootHash
	}
	postion++
	//checkPledge
	if srtAmount, amount, duration, burnSRTAddress, ok := snap.StorageData.checkSRentPg(currentSRentPg, sRentPg, txSender, snap.RevenueStorage, snap.SystemConfig.ExchRate); ok {
		sRentPg.BurnSRTAmount = srtAmount
		sRentPg.BurnAmount = amount
		sRentPg.Duration = duration
		sRentPg.BurnSRTAddress = burnSRTAddress

		if !snap.SRTIndex.checkEnoughSRTPg(currentSRentPg, sRentPg, number-1, a.db) {
			log.Warn("sRent", "checkEnoughSRT fail", sRentPg.BurnSRTAddress)
			return currentSRentPg
		}
		if state.GetBalance(txSender).Cmp(amount) < 0 {
			log.Warn("sRentReNewPg", "balance", state.GetBalance(txSender))
			return currentSRentPg
		}
		state.SetBalance(txSender, new(big.Int).Sub(state.GetBalance(txSender), amount))
		topics := make([]common.Hash, 2)
		topics[0].UnmarshalText([]byte("0xf145aaf8213a13521c09380bc80e9f77d4aa86f181a31bdf688f4693e95b6647"))
		topics[1].SetBytes(sRentPg.Hash.Bytes())
		dataList := make([]common.Hash, 3)
		dataList[0].SetBytes(sRentPg.Address.Bytes())
		dataList[1].SetBytes(sRentPg.Capacity.Bytes())
		dataList[2].SetBytes(sRentPg.RootHash.Bytes())
		data := dataList[0].Bytes()
		data = append(data, dataList[1].Bytes()...)
		a.addCustomerTxLog(tx, receipts, topics, data)
		currentSRentPg = append(currentSRentPg, sRentPg)
	} else {
		log.Warn("sRentPg", "checkSRentPg fail", sRentPg.Hash)
	}
	return currentSRentPg
}
func (a *Alien) processLeaseRenewal(currentSRentReNew []LeaseRenewalRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot, number uint64) []LeaseRenewalRecord {
	if len(txDataInfo) < 6 {
		log.Warn("sRentReNew", "parameter number", len(txDataInfo))
		return currentSRentReNew
	}
	sRentReNew := LeaseRenewalRecord{
		Address:  common.Address{},
		Hash:     common.Hash{},
		Duration: big.NewInt(0),
		Price:    big.NewInt(0),
		Tenant:   common.Address{},
		NewHash:  common.Hash{},
		Capacity: big.NewInt(0),
	}
	postion := 3
	if err := sRentReNew.Address.UnmarshalText1([]byte(txDataInfo[postion])); err != nil {
		log.Warn("sRentReNew", "Hash", txDataInfo[postion])
		return currentSRentReNew
	}
	postion++
	sRentReNew.Hash = common.HexToHash(txDataInfo[postion])
	postion++
	if duration, err := strconv.ParseUint(txDataInfo[postion], 10, 32); err != nil {
		log.Warn("sRentReNew", "duration", txDataInfo[postion])
		return currentSRentReNew
	} else {
		sRentReNew.Duration = new(big.Int).SetUint64(duration)
	}
	if sRentReNew.Duration.Cmp(snap.SystemConfig.Deposit[sscEnumMinimumRent]) < 0 {
		log.Warn("sRentReNew", "Duration", sRentReNew.Duration)
		return currentSRentReNew
	}
	if tenant, ok := snap.StorageData.checkSRentReNew(currentSRentReNew, sRentReNew, txSender, number, a.blockPerDay()); ok {
		sRentReNew.Tenant = tenant
	} else {
		log.Warn("sRentReNew", "checkSRentReNew fail", sRentReNew.Hash)
		return currentSRentReNew
	}
	lease := snap.StorageData.StoragePledge[sRentReNew.Address].Lease
	l := lease[sRentReNew.Hash]
	sRentReNew.Price = l.UnitPrice
	sRentReNew.Capacity = l.Capacity
	if !snap.SRTIndex.checkEnoughSRTReNew(currentSRentReNew, sRentReNew, number-1, a.db) {
		log.Warn("sRentReNew", "checkEnoughSRT fail", sRentReNew.Tenant)
		return currentSRentReNew
	}
	sRentReNew.NewHash = tx.Hash()
	topics := make([]common.Hash, 2)
	topics[0].UnmarshalText([]byte("0xad3545265bff0a514f14821359a92d5b238073e1058ef0f7d83cd3ddcc7306cb")) //web3.sha3("stReNew(address)")
	topics[1].SetBytes(sRentReNew.Hash.Bytes())
	a.addCustomerTxLog(tx, receipts, topics, nil)
	currentSRentReNew = append(currentSRentReNew, sRentReNew)
	return currentSRentReNew
}
func (a *Alien) processLeaseRenewalPledge(currentSRentReNewPg []LeaseRenewalPledgeRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot, number uint64) []LeaseRenewalPledgeRecord {
	if len(txDataInfo) < 7 {
		log.Warn("sRentReNewPg", "parameter number", len(txDataInfo))
		return currentSRentReNewPg
	}
	sRentPg := LeaseRenewalPledgeRecord{
		Address:    common.Address{},
		Hash:       common.Hash{},
		Capacity:   big.NewInt(0),
		RootHash:   common.Hash{},
		Duration:   big.NewInt(0),
		PledgeHash: tx.Hash(),
	}
	postion := 3
	if err := sRentPg.Address.UnmarshalText1([]byte(txDataInfo[postion])); err != nil {
		log.Warn("sRentReNewPg", "Hash", txDataInfo[postion])
		return currentSRentReNewPg
	}
	postion++
	sRentPg.Hash = common.HexToHash(txDataInfo[postion])
	postion++
	if capacity, err := decimal.NewFromString(txDataInfo[postion]); err != nil {
		log.Warn("sRentReNewPg", "Capacity", txDataInfo[postion])
		return currentSRentReNewPg
	} else {
		sRentPg.Capacity = capacity.BigInt()
	}
	postion++
	if rootHash, ok := snap.StorageData.verifyParamsStoragePoc(txDataInfo, postion, tx.Nonce()); !ok {
		log.Warn("sRentReNewPg verify fail", " RootHash", rootHash)
		return currentSRentReNewPg
	} else {
		sRentPg.RootHash = rootHash
	}
	postion++
	//checkPledge
	if srtAmount, amount, duration, burnSRTAddress, ok := snap.StorageData.checkSRentReNewPg(currentSRentReNewPg, sRentPg, txSender, snap.RevenueStorage, snap.SystemConfig.ExchRate); ok {
		sRentPg.BurnSRTAmount = srtAmount
		sRentPg.BurnAmount = amount
		sRentPg.Duration = duration
		sRentPg.BurnSRTAddress = burnSRTAddress
		if state.GetBalance(txSender).Cmp(amount) < 0 {
			log.Warn("sRentReNewPg", "balance", state.GetBalance(txSender))
			return currentSRentReNewPg
		}
		if !snap.SRTIndex.checkEnoughSRTReNewPg(currentSRentReNewPg, sRentPg, number-1, a.db) {
			log.Warn("sRentReNewPg", "checkEnoughSRT fail", sRentPg.BurnSRTAddress)
			return currentSRentReNewPg
		}
		state.SetBalance(txSender, new(big.Int).Sub(state.GetBalance(txSender), amount))
		topics := make([]common.Hash, 2)
		topics[0].UnmarshalText([]byte("0x24461fc75f60084c7cefe35795e6365d21728afd90a7eee606bac1f92013baec"))
		topics[1].SetBytes(sRentPg.Hash.Bytes())
		dataList := make([]common.Hash, 3)
		dataList[0].SetBytes(sRentPg.Address.Bytes())
		dataList[1].SetBytes(sRentPg.Capacity.Bytes())
		dataList[2].SetBytes(sRentPg.RootHash.Bytes())
		data := dataList[0].Bytes()
		data = append(data, dataList[1].Bytes()...)
		a.addCustomerTxLog(tx, receipts, topics, data)
		currentSRentReNewPg = append(currentSRentReNewPg, sRentPg)
	} else {
		log.Warn("sRentReNewPg", "checkSRentReNewPg fail", sRentPg.Hash)
	}
	return currentSRentReNewPg
}

func (a *Alien) processLeaseRescind(currentSRescind []LeaseRescindRecord, currentExchangeSRT []ExchangeSRTRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot, number uint64) ([]LeaseRescindRecord, []ExchangeSRTRecord) {
	if len(txDataInfo) < 5 {
		log.Warn("stRescind", "parameter number", len(txDataInfo))
		return currentSRescind, currentExchangeSRT
	}
	sRescind := LeaseRescindRecord{
		Address: common.Address{},
		Hash:    common.Hash{},
	}
	postion := 3
	if err := sRescind.Address.UnmarshalText1([]byte(txDataInfo[postion])); err != nil {
		log.Warn("stRescind", "Hash", txDataInfo[postion])
		return currentSRescind, currentExchangeSRT
	}
	postion++
	sRescind.Hash = common.HexToHash(txDataInfo[postion])
	//checkSRescind
	if ok := snap.StorageData.checkSRescind(currentSRescind, sRescind, txSender, snap.SystemConfig.ExchRate, number, a.blockPerDay()); ok {
		topics := make([]common.Hash, 2)
		topics[0].UnmarshalText([]byte("0x3bfad54852baf2b8be1ae9452a2b1d07e9c03e139b622817417852cc78d06100"))
		topics[1].SetBytes(sRescind.Hash.Bytes())
		a.addCustomerTxLog(tx, receipts, topics, nil)
		currentSRescind = append(currentSRescind, sRescind)
	} else {
		log.Warn("stRescind", "checkSRescind fail", sRescind.Hash)
	}
	return currentSRescind, currentExchangeSRT
}

func (s *StorageData) checkSRescind(currentSRescind []LeaseRescindRecord, sRescind LeaseRescindRecord, txSender common.Address, exchRate uint32, number uint64, blockPerDay uint64) bool {
	for _, item := range currentSRescind {
		if item.Hash == sRescind.Hash {
			return false
		}
	}
	if _, ok := s.StoragePledge[sRescind.Address]; !ok {
		return false
	}
	if _, ok := s.StoragePledge[sRescind.Address].Lease[sRescind.Hash]; !ok {
		return false
	}
	lease := s.StoragePledge[sRescind.Address].Lease[sRescind.Hash]
	if lease.Address != txSender {
		return false
	}
	status := lease.Status
	if status != LeaseBreach {
		return false
	}
	return true
}

func (s *StorageData) updateLeaseRescind(sRescinds []LeaseRescindRecord, number *big.Int, db ethdb.Database) {
	for _, sRescind := range sRescinds {
		if _, ok := s.StoragePledge[sRescind.Address]; !ok {
			continue
		}
		if _, ok := s.StoragePledge[sRescind.Address].Lease[sRescind.Hash]; !ok {
			continue
		}
		lease := s.StoragePledge[sRescind.Address].Lease[sRescind.Hash]
		lease.Status = LeaseUserRescind
		s.accumulateLeaseHash(sRescind.Address, lease)
	}
	s.accumulateHeaderHash()
}

func (s *StorageData) storageVerificationCheck(number uint64, blockPerday uint64, passTime *big.Int, rate uint32, revenueStorage map[common.Address]*RevenueParameter, period uint64, db ethdb.Database, basePrice *big.Int,currentLockReward [] LockRewardRecord) ([] LockRewardRecord,[]ExchangeSRTRecord, *big.Int,error) {

	sussSPAddrs, sussRentHashs, storageRatios := s.storageVerify(number, blockPerday, revenueStorage)

	err:=s.saveSPledgeSuccTodb(sussSPAddrs, db, number)
	if err!=nil{
		return currentLockReward,nil, nil,err
	}
	err=s.saveRentSuccTodb(sussRentHashs, db, number)
	if err!=nil{
		return currentLockReward,nil, nil,err
	}
	revertSpaceLockReward, revertExchangeSRT := s.dealLeaseStatus(number, blockPerday, rate, blockPerday)
	err=s.saveRevertSpaceLockRewardTodb(revertSpaceLockReward, db, number)
	if err!=nil{
		return currentLockReward,nil, nil,err
	}
	err=s.saveRevertExchangeSRTTodb(revertExchangeSRT, db, number)
	if err!=nil{
		return currentLockReward,nil, nil,err
	}
	storageRatios = s.calcStorageRatio(storageRatios)
	err=s.saveStorageRatiosTodb(storageRatios, db, number)
	if err!=nil{
		return currentLockReward,nil, nil,err
	}
	harvest := big.NewInt(0)
	zero := big.NewInt(0)
	spaceLockReward, spaceHarvest := s.calcStoragePledgeReward(storageRatios, revenueStorage, number, period)
	if spaceHarvest.Cmp(zero) > 0 {
		harvest = new(big.Int).Add(harvest, spaceHarvest)
	}
	err=s.saveSpaceLockRewardTodb(spaceLockReward, revenueStorage, db, number)
	if err!=nil{
		return currentLockReward,nil, nil,err
	}
	s.deletePasstimeLease(number, blockPerday, passTime)
	LockLeaseReward, leaseHarvest := s.accumulateLeaseRewards(number, blockPerday, storageRatios, sussRentHashs, basePrice, revenueStorage)
	if leaseHarvest.Cmp(zero) > 0 {
		harvest = new(big.Int).Add(harvest, leaseHarvest)
	}
	err=s.saveLeaseLockRewardTodb(LockLeaseReward, db, number)
	if err!=nil{
		return currentLockReward,nil, nil,err
	}
	if  currentLockReward!= nil{
		for _,item:= range revertSpaceLockReward{
			currentLockReward=append(currentLockReward,LockRewardRecord{
				Target:  item.Target,
				Amount :item.Amount,
				IsReward :sscEnumBandwidthReward,
			})
		}
		for _,item:= range spaceLockReward{
			currentLockReward=append(currentLockReward,LockRewardRecord{
				Target:  item.Target,
				Amount :item.Amount,
				IsReward :sscEnumBandwidthReward,
			})
		}

		for _,item:= range LockLeaseReward{
			currentLockReward=append(currentLockReward,LockRewardRecord{
				Target:  item.Target,
				Amount :item.Amount,
				IsReward :sscEnumFlwReward,
			})
		}
	}
	return currentLockReward,revertExchangeSRT, harvest,nil
}

/**
 *Storage space recovery certificate
 */
func (a *Alien) storageRecoveryCertificate(storageRecoveryData []SPledgeRecoveryRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot, blocknumber *big.Int, chain consensus.ChainHeaderReader) []SPledgeRecoveryRecord {
	log.Info("storageRecoveryCertificate", "txDataInfo", txDataInfo)
	if len(txDataInfo) < 7 {
		log.Warn("storage Recovery Certificate", "parameter error", len(txDataInfo))
		return storageRecoveryData
	}
	pledgeAddr := common.HexToAddress(txDataInfo[3])
	if pledgeAddr != txSender {
		if revenue, ok := snap.RevenueStorage[pledgeAddr]; !ok || revenue.RevenueAddress != txSender {
			log.Warn("storage Recovery Certificate  no role", " txSender", txSender)
			return storageRecoveryData
		}
	}
	storagepledge := snap.StorageData.StoragePledge[pledgeAddr]
	if storagepledge == nil {
		log.Warn("storage Recovery Certificate  not find pledge", " pledgeAddr", pledgeAddr)
		return storageRecoveryData
	}

	validData := txDataInfo[6]
	verifydatas := strings.Split(validData, ",")
	if len(verifydatas) < 10 {
		log.Warn("verifyStoragePoc", "invalide poc string format")
		return storageRecoveryData
	}
	rootHash := verifydatas[len(verifydatas)-1]
	//verifyNumber,_:=decimal.NewFromString(verifydatas[0])
	verifyHeader := chain.GetHeaderByHash(common.HexToHash(verifydatas[2]))
	if verifyHeader == nil || verifyHeader.Number.String() != verifydatas[0] || strconv.FormatInt(int64(verifyHeader.Nonce.Uint64()), 10) != verifydatas[1] {
		log.Warn("storageRecoveryCertificate  GetHeaderByHash not find by hash  ", "verifydatas", verifydatas)
		return storageRecoveryData
	}
	if !verifyStoragePoc(validData, rootHash, verifyHeader.Nonce.Uint64()) {
		log.Warn("storage  Recovery Certificate   valid  faild", "validData", validData)
		return storageRecoveryData
	}
	leaseHashStr := strings.Split(txDataInfo[4], ",")
	currNumber := big.NewInt(int64(snap.Number))
	var delLeaseHash []common.Hash
	for _, hashStr := range leaseHashStr {
		leaseHash := common.HexToHash(hashStr)
		if lease, ok := storagepledge.Lease[leaseHash]; ok {
			if lease.Status == LeaseReturn {
				delLeaseHash = append(delLeaseHash, leaseHash)
			}
		}
	}
	if len(delLeaseHash) != len(leaseHashStr) {
		log.Warn("storage  Recovery Certificate  There are leases that have not expired ", " leaseHash", txDataInfo[4])
		return storageRecoveryData
	}
	storageCapacity, err := decimal.NewFromString(txDataInfo[5])
	if err != nil {
		log.Warn("storage  Recovery storageCapacity  format err ", " storageCapacity", txDataInfo[5])
		return storageRecoveryData
	}
	totalcapacity := storagepledge.TotalCapacity
	if storageCapacity.BigInt().Cmp(totalcapacity) > 0 {
		log.Warn("storage  Recovery storageCapacity more than totalcapacity", " storageCapacity", txDataInfo[5])
		return storageRecoveryData
	}
	storageRecoveryData = append(storageRecoveryData, SPledgeRecoveryRecord{
		Address:       pledgeAddr,
		LeaseHash:     delLeaseHash,
		SpaceCapacity: storageCapacity.BigInt(),
		RootHash:      common.HexToHash(rootHash),
		ValidNumber:   currNumber,
	})
	topics := make([]common.Hash, 3)
	topics[0].UnmarshalText([]byte("0Xf145aaf8213a13521c09380bc80e9f77d4aa86f181a31bdf684532e95b6647"))
	topics[1].SetBytes(pledgeAddr.Bytes())
	topics[2].SetBytes([]byte(storageCapacity.String()))
	a.addCustomerTxLog(tx, receipts, topics, nil)
	return storageRecoveryData
}
func (s *Snapshot) updateStorageRecoveryData(storageRecoveryData []SPledgeRecoveryRecord, headerNumber *big.Int, db ethdb.Database) {
	if storageRecoveryData == nil || len(storageRecoveryData) == 0 {
		return
	}
	for _, storageRvdata := range storageRecoveryData {

		if pledgeData, ok := s.StorageData.StoragePledge[storageRvdata.Address]; ok {
			for _, leaseHash := range storageRvdata.LeaseHash {
				delete(pledgeData.Lease, leaseHash)
			}
			delete(pledgeData.StorageSpaces.StorageFile, pledgeData.StorageSpaces.RootHash)
			pledgeData.StorageSpaces.RootHash = storageRvdata.RootHash
			pledgeData.StorageSpaces.StorageFile[storageRvdata.RootHash] = &StorageFile{
				Capacity:                    storageRvdata.SpaceCapacity,
				CreateTime:                  storageRvdata.ValidNumber,
				LastVerificationTime:        storageRvdata.ValidNumber,
				LastVerificationSuccessTime: storageRvdata.ValidNumber,
				ValidationFailureTotalTime:  big.NewInt(0),
			}
			pledgeData.StorageSpaces.StorageCapacity = storageRvdata.SpaceCapacity
			pledgeData.StorageSpaces.ValidationFailureTotalTime = big.NewInt(0)
			pledgeData.StorageSpaces.LastVerificationSuccessTime = storageRvdata.ValidNumber
			pledgeData.StorageSpaces.LastVerificationTime = storageRvdata.ValidNumber
			s.StorageData.accumulatePledgeHash(storageRvdata.Address)
		}
	}
	s.StorageData.accumulateHeaderHash()

}

func (a *Alien) applyStorageProof(storageProofRecord []StorageProofRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot, blocknumber *big.Int, chain consensus.ChainHeaderReader) []StorageProofRecord {
	//log.Debug("applyStorageProof", "txDataInfo", txDataInfo)
	if len(txDataInfo) < 7 {
		log.Warn("Storage Proof", "parameter error", len(txDataInfo))
		return storageProofRecord
	}
	pledgeAddr := common.HexToAddress(txDataInfo[3])
	if pledgeAddr != txSender {
		log.Warn("Storage Proof txSender no role", " txSender", txSender, "pledgeAddr", pledgeAddr)
		return storageProofRecord

	}
	storagepledge := snap.StorageData.StoragePledge[pledgeAddr]
	if storagepledge == nil {
		log.Warn("Storage Proof not find pledge", " pledgeAddr", pledgeAddr)
		return storageProofRecord
	}
	var capacity *big.Int
	if capvalue, err := decimal.NewFromString(txDataInfo[5]); err != nil {
		log.Warn("Storage Proof capvalue format error", "Capacity", txDataInfo[5])
		return storageProofRecord
	} else {
		capacity = capvalue.BigInt()
	}
	var tragetCapacity *big.Int
	validData := txDataInfo[6]
	verifydatas := strings.Split(validData, ",")
	rootHash := common.HexToHash(verifydatas[len(verifydatas)-1])
	leaseHash := common.Hash{}
	currNumber := big.NewInt(int64(snap.Number))
	if len(txDataInfo[4]) > 10 {
		leaseHash = common.HexToHash(txDataInfo[4])
		if _, ok := storagepledge.Lease[leaseHash]; !ok {
			log.Warn("Storage Proof not find leaseHash", " leaseHash", leaseHash)
			return storageProofRecord
		}
		storageFile := storagepledge.Lease[leaseHash].StorageFile
		if _, ok := storageFile[rootHash]; !ok {
			log.Warn("Storage Proof lease not find rootHash", " rootHash", rootHash)
			return storageProofRecord
		}
		lease := storagepledge.Lease[leaseHash]
		tragetCapacity = lease.Capacity
	} else {
		storageFile := storagepledge.StorageSpaces.StorageFile
		if _, ok := storageFile[rootHash]; !ok {
			log.Warn("applyStorageProof not find rootHash", " rootHash", rootHash)
			return storageProofRecord
		}
		tragetCapacity = storageFile[rootHash].Capacity
	}
	if tragetCapacity == nil || tragetCapacity.Cmp(capacity) != 0 {
		log.Warn("applyStorageProof  capacity not same", " capacity", capacity)
		return storageProofRecord
	}
	pocs := strings.Split(validData, ",")
	if len(pocs) < 10 {
		log.Warn("verifyStoragePoc", "invalide poc string format")
		return storageProofRecord
	}
	verifyHeader := chain.GetHeaderByHash(common.HexToHash(pocs[2]))
	if verifyHeader == nil || verifyHeader.Number.String() != pocs[0] || strconv.FormatInt(int64(verifyHeader.Nonce.Uint64()), 10) != pocs[1] {
		log.Warn("applyStorageProof  GetHeaderByHash not find by hash  ", "poc", pocs)
		return storageProofRecord
	}
	if currNumber.Cmp(new(big.Int).Add(proofTimeOut,verifyHeader.Number)) > 0{
		log.Warn("applyStorageProof data timeout  ", "TimeOut", proofTimeOut,"currNumber",currNumber,"proof number",verifyHeader.Number)
		return storageProofRecord
	}
	if !verifyStoragePoc(validData, storagepledge.StorageSpaces.RootHash.String(), 0) {
		log.Warn("applyStorageProof   verify  faild", "roothash", storagepledge.StorageSpaces.RootHash.String())
        return storageProofRecord
	}

	proofRecord := StorageProofRecord{
		Address:                     pledgeAddr,
		RootHash:                    rootHash,
		LeaseHash:                   leaseHash,
		LastVerificationTime:        currNumber,
		LastVerificationSuccessTime: currNumber,
	}
	topics := make([]common.Hash, 3)
	topics[0].UnmarshalText([]byte("0xb259d26eb65071ded303add129ecef7af12cf17a8ea9d41f7ff0cfa5af3123f8"))
	topics[1].SetBytes(pledgeAddr.Bytes())
	topics[2].SetBytes([]byte(currNumber.String()))
	a.addCustomerTxLog(tx, receipts, topics, nil)
	storageProofRecord = append(storageProofRecord, proofRecord)
	return storageProofRecord
}
func (s *Snapshot) updateStorageProof(proofDatas []StorageProofRecord, headerNumber *big.Int, db ethdb.Database) {
	if proofDatas == nil || len(proofDatas) == 0 {
		return
	}
	nilHash := common.Hash{}
	for _, proof := range proofDatas {
		storagePledge := s.StorageData.StoragePledge[proof.Address]
		if proof.LeaseHash == nilHash {
			storagePledge.StorageSpaces.StorageFile[proof.RootHash].LastVerificationSuccessTime = proof.LastVerificationSuccessTime
			storagePledge.StorageSpaces.StorageFile[proof.RootHash].LastVerificationTime = proof.LastVerificationTime
			s.StorageData.accumulateSpaceStorageFileHash(proof.Address, storagePledge.StorageSpaces.StorageFile[proof.RootHash])
		} else {
			storagePledge.Lease[proof.LeaseHash].StorageFile[proof.RootHash].LastVerificationTime = proof.LastVerificationTime
			storagePledge.Lease[proof.LeaseHash].StorageFile[proof.RootHash].LastVerificationSuccessTime = proof.LastVerificationSuccessTime
			s.StorageData.accumulateLeaseStorageFileHash(proof.Address, proof.LeaseHash, storagePledge.Lease[proof.LeaseHash].StorageFile[proof.RootHash])
		}

	}
	s.StorageData.accumulateHeaderHash()
}

func (s *StorageData) calStorageLeaseReward(capacity decimal.Decimal, bandwidthIndex decimal.Decimal, storageIndex decimal.Decimal,
	priceIndex decimal.Decimal, duration decimal.Decimal, headerNumber uint64, blockPreDay uint64) *big.Int {
	yearBlockNumber := 365 * blockPreDay
	n := headerNumber / yearBlockNumber
	if headerNumber%yearBlockNumber > 0 {
		n += 1
	}
	ebReward := decimal.NewFromBigInt(totalBlockReward, 0).Mul(decimal.NewFromFloat(float64(1) - math.Pow(float64(0.5), float64(n)/float64(12))))
	tbUTGRate := ebReward.Div(decimal.NewFromInt(1048576))
	return capacity.Mul(priceIndex).Mul(duration).Mul(bandwidthIndex).Mul(storageIndex).Mul(tbUTGRate).BigInt()
}

func (s *StorageData) accumulateLeaseRewards(headerNumber uint64, blockPerDay uint64, ratios map[common.Address]*StorageRatio,
	addrs []common.Hash, basePrice *big.Int, revenueStorage map[common.Address]*RevenueParameter) ([]SpaceRewardRecord, *big.Int) {
	var LockReward []SpaceRewardRecord
	//basePrice := // SRT /TB.day
	storageHarvest := big.NewInt(0)
	if nil == addrs || len(addrs) == 0 {
		return LockReward, storageHarvest
	}
	totalLeaseSpace := decimal.NewFromInt(0)
	validSuccLesae := make(map[common.Hash]uint64)
	for _, leaseHash := range addrs {
		validSuccLesae[leaseHash] = 1
	}
	for _, storage := range s.StoragePledge {
		for leaseHash, lease := range storage.Lease {
			if _, ok := validSuccLesae[leaseHash]; ok {
				totalLeaseSpace = totalLeaseSpace.Add(decimal.NewFromBigInt(lease.Capacity, 0).Div(decimal.NewFromInt(1099511627776))) //TB
			}
		}
	}

	for pledgeAddr, storage := range s.StoragePledge {
		totalReward := big.NewInt(0)
		bandwidthIndex := getBandwaith(storage.Bandwidth)
		if revenue, ok := revenueStorage[pledgeAddr]; ok {
			for leaseHash, lease := range storage.Lease {
				if _, ok2 := validSuccLesae[leaseHash]; ok2 {
					leaseCapacity := decimal.NewFromBigInt(lease.Capacity, 0).Div(decimal.NewFromInt(1099511627776)) //to TB
					toTBprice := new(big.Int).Mul(lease.UnitPrice, big.NewInt(1024))                                 //SRT /TB.day
					priceIndex := decimal.NewFromBigInt(toTBprice, 0).Div(decimal.NewFromBigInt(basePrice, 0))
					if _, ok3 := ratios[revenue.RevenueAddress]; ok3 {
						reward := s.calStorageLeaseReward(leaseCapacity, bandwidthIndex, ratios[revenue.RevenueAddress].Ratio, priceIndex, decimal.NewFromBigInt(lease.Duration, 0),
							headerNumber, blockPerDay)
						totalReward = new(big.Int).Add(totalReward, reward)
					}
				}
			}
			if totalReward.Cmp(big.NewInt(0)) > 0 {
				LockReward = append(LockReward, SpaceRewardRecord{
					Target:  pledgeAddr,
					Amount:  totalReward,
					Revenue: revenue.RevenueAddress,
				})
				storageHarvest = new(big.Int).Add(storageHarvest, totalReward)
			}
		}
	}
	return LockReward, storageHarvest
}

func getBandwaith(bandwidth *big.Int) decimal.Decimal {
	if bandwidth.Cmp(big.NewInt(29)) <= 0 {
		return decimal.NewFromInt(0)
	}
	if bandwidth.Cmp(big.NewInt(30)) >= 0 && bandwidth.Cmp(big.NewInt(50)) <= 0 {
		return decimal.NewFromFloat(0.7)
	}
	if bandwidth.Cmp(big.NewInt(51)) >= 0 && bandwidth.Cmp(big.NewInt(99)) <= 0 {
		return decimal.NewFromFloat(0.9)
	}
	if bandwidth.Cmp(big.NewInt(100)) == 0 {
		return decimal.NewFromFloat(1)
	}
	if bandwidth.Cmp(big.NewInt(101)) >= 0 && bandwidth.Cmp(big.NewInt(500)) <= 0 {
		return decimal.NewFromFloat(1.1)
	}
	if bandwidth.Cmp(big.NewInt(501)) >= 0 && bandwidth.Cmp(big.NewInt(1023)) <= 0 {
		return decimal.NewFromFloat(1.3)
	}
	return decimal.NewFromFloat(1.5)

}

func (s *StorageData) nYearSpaceProfitReward(n float64) decimal.Decimal {
	onecut := float64(1) - math.Pow(float64(0.5), n/float64(3))
	yearScale := decimal.NewFromFloat(onecut)
	yearReward := yearScale.Mul(decimal.NewFromBigInt(totalSpaceProfitReward, 0))
	return yearReward
}

func (s *StorageData) checkSRentReNewPg(currentSRentReNewPg []LeaseRenewalPledgeRecord, sRentReNewPg LeaseRenewalPledgeRecord, txSender common.Address, revenueStorage map[common.Address]*RevenueParameter, exchRate uint32) (*big.Int, *big.Int, *big.Int, common.Address, bool) {
	nilHash := common.Address{}
	for _, item := range currentSRentReNewPg {
		if item.Address == sRentReNewPg.Address {
			return nil, nil, nil, nilHash, false
		}
	}
	//checkCapacity
	if _, ok := s.StoragePledge[sRentReNewPg.Address]; !ok {
		return nil, nil, nil, nilHash, false
	}
	if _, ok := s.StoragePledge[sRentReNewPg.Address].Lease[sRentReNewPg.Hash]; !ok {
		return nil, nil, nil, nilHash, false
	}
	lease := s.StoragePledge[sRentReNewPg.Address].Lease[sRentReNewPg.Hash]
	if lease.Capacity.Cmp(sRentReNewPg.Capacity) != 0 {
		return nil, nil, nil, nilHash, false
	}
	//checkowner

	if lease.DepositAddress != txSender {
		return nil, nil, nil, nilHash, false
	}

	//checkfileproof  todo
	hasRent := false
	duration := big.NewInt(0)
	unitPrice := lease.UnitPrice
	for _, detail := range lease.LeaseList {
		if detail.Deposit.Cmp(big.NewInt(0)) <= 0 {
			hasRent = true
			duration = detail.Duration
		}
	}
	if !hasRent {
		return nil, nil, nil, nilHash, false
	}
	//Calculate the pledge deposit
	srtAmount := new(big.Int).Mul(duration, unitPrice)
	srtAmount = new(big.Int).Mul(srtAmount, lease.Capacity)
	srtAmount = new(big.Int).Div(srtAmount, gbTob)
	amount := new(big.Int).Div(new(big.Int).Mul(srtAmount, big.NewInt(10000)), big.NewInt(int64(exchRate)))
	return srtAmount, amount, duration, lease.Address, true
}

func (a *Alien) exchangeStoragePrice(storageExchangePriceRecord []StorageExchangePriceRecord, txDataInfo []string, txSender common.Address, tx *types.Transaction, receipts []*types.Receipt, state *state.StateDB, snap *Snapshot, blocknumber *big.Int) []StorageExchangePriceRecord {
	if len(txDataInfo) < 5 {
		log.Warn("exchange   Price  of Storage", "parameter error", len(txDataInfo))
		return storageExchangePriceRecord
	}
	pledgeAddr := common.HexToAddress(txDataInfo[3])
	if pledgeAddr != txSender {
		if revenue, ok := snap.RevenueStorage[pledgeAddr]; !ok || revenue.RevenueAddress != txSender {
			log.Warn("exchange   Price  of Storage  [no role]", " txSender", txSender)
			return storageExchangePriceRecord
		}
	}
	if _, ok := snap.StorageData.StoragePledge[pledgeAddr]; !ok {
		log.Warn("exchange  Price not find Pledge", " pledgeAddr", pledgeAddr)
		return storageExchangePriceRecord
	}
	price, _ := decimal.NewFromString(txDataInfo[4])
	basePrice := snap.SystemConfig.Deposit[sscEnumStoragePrice]
	if price.BigInt().Cmp(basePrice) < 0 || price.BigInt().Cmp(new(big.Int).Mul(big.NewInt(10), basePrice)) > 0 {
		log.Warn("exchange  Price not legal", " pledgeAddr", pledgeAddr, "price", price, "basePrice", basePrice)
		return storageExchangePriceRecord
	}
	storageExchangePriceRecord = append(storageExchangePriceRecord, StorageExchangePriceRecord{
		Address: pledgeAddr,
		Price:   price.BigInt(),
	})
	topics := make([]common.Hash, 3)
	topics[0].UnmarshalText([]byte("0xb12bf5b909b60bb08c3e990dcb437a238072a91629c666541b667da82b3ee49b"))
	topics[1].SetBytes(pledgeAddr.Bytes())
	topics[2].SetBytes([]byte(txDataInfo[4]))
	a.addCustomerTxLog(tx, receipts, topics, nil)
	return storageExchangePriceRecord
}

func (s *Snapshot) updateStoragePrice(storageExchangePriceRecord []StorageExchangePriceRecord, headerNumber *big.Int, db ethdb.Database) {
	if storageExchangePriceRecord == nil || len(storageExchangePriceRecord) == 0 {
		return
	}
	for _, exchangeprice := range storageExchangePriceRecord {
		if _, ok := s.StorageData.StoragePledge[exchangeprice.Address]; ok {
			s.StorageData.StoragePledge[exchangeprice.Address].Price = exchangeprice.Price
		}
	}
}

func (s *StorageData) updateLeaseRenewalPledge(pg []LeaseRenewalPledgeRecord, number *big.Int, db ethdb.Database, blockPerday uint64) {
	for _, sRentPg := range pg {
		if _, ok := s.StoragePledge[sRentPg.Address]; !ok {
			continue
		}
		if _, ok := s.StoragePledge[sRentPg.Address].Lease[sRentPg.Hash]; !ok {
			continue
		}
		lease := s.StoragePledge[sRentPg.Address].Lease[sRentPg.Hash]
		lease.RootHash = sRentPg.RootHash
		lease.Deposit = new(big.Int).Add(lease.Deposit, sRentPg.BurnAmount)
		lease.Cost = new(big.Int).Add(lease.Cost, sRentPg.BurnSRTAmount)
		lease.Duration = new(big.Int).Add(lease.Duration, sRentPg.Duration)
		if _, ok := lease.StorageFile[sRentPg.RootHash]; !ok {
			lease.StorageFile[sRentPg.RootHash] = &StorageFile{
				Capacity:                    lease.Capacity,
				CreateTime:                  number,
				LastVerificationTime:        number,
				LastVerificationSuccessTime: number,
				ValidationFailureTotalTime:  big.NewInt(0),
			}
			s.accumulateLeaseStorageFileHash(sRentPg.Address, sRentPg.Hash, lease.StorageFile[sRentPg.RootHash])
		}
		startTime := big.NewInt(0)
		duration := big.NewInt(0)
		for _, leaseDetail := range lease.LeaseList {
			if leaseDetail.Deposit.Cmp(big.NewInt(0)) > 0 && leaseDetail.StartTime.Cmp(startTime) > 0 {
				startTime = leaseDetail.StartTime
				duration = new(big.Int).Mul(leaseDetail.Duration, new(big.Int).SetUint64(blockPerday))
			}
		}
		startTime = new(big.Int).Add(startTime, duration)
		startTime = new(big.Int).Add(startTime, big.NewInt(1))
		for _, detail := range lease.LeaseList {
			if detail.Deposit.Cmp(big.NewInt(0)) == 0 {
				detail.Cost = new(big.Int).Add(detail.Cost, sRentPg.BurnSRTAmount)
				detail.Deposit = new(big.Int).Add(detail.Deposit, sRentPg.BurnAmount)
				detail.PledgeHash = sRentPg.PledgeHash
				detail.StartTime = startTime
				s.accumulateLeaseDetailHash(sRentPg.Address, sRentPg.Hash, detail)
				break
			}
		}
		lease.Status = LeaseNormal
		s.accumulateLeaseHash(sRentPg.Address, lease)
	}
	s.accumulateHeaderHash()
}

func (s *StorageData) accumulateSpaceStorageFileHash(pledgeAddr common.Address, storagefile *StorageFile) common.Hash {
	storagefile.Hash = getHash(storagefile.LastVerificationTime.String() + storagefile.LastVerificationSuccessTime.String() +
		storagefile.ValidationFailureTotalTime.String() + storagefile.Capacity.String() + storagefile.CreateTime.String())
	s.accumulateSpaceHash(pledgeAddr)
	return storagefile.Hash
}

func (s *StorageData) accumulateLeaseStorageFileHash(pledgeAddr common.Address, leaseKey common.Hash, storagefile *StorageFile) {
	storagePledge := s.StoragePledge[pledgeAddr]
	lease := storagePledge.Lease[leaseKey]
	storagefile.Hash = getHash(storagefile.LastVerificationTime.String() + storagefile.LastVerificationSuccessTime.String() +
		storagefile.ValidationFailureTotalTime.String() + storagefile.Capacity.String() + storagefile.CreateTime.String())
	s.accumulateLeaseHash(pledgeAddr, lease)
}
func (s *StorageData) accumulateLeaseDetailHash(pledgeAddr common.Address, leaseKey common.Hash, leasedetail *LeaseDetail) {
	storagePledge := s.StoragePledge[pledgeAddr]
	lease := storagePledge.Lease[leaseKey]
	leasedetail.Hash = getHash(leasedetail.ValidationFailureTotalTime.String() + leasedetail.Duration.String() + leasedetail.Cost.String() +
		leasedetail.Deposit.String() + leasedetail.StartTime.String() + leasedetail.PledgeHash.String() + leasedetail.RequestHash.String() + leasedetail.RequestTime.String() +
		strconv.Itoa(leasedetail.Revert))
	s.accumulateLeaseHash(pledgeAddr, lease)
}
func (s *StorageData) accumulateLeaseHash(pledgeAddr common.Address, lease *Lease) common.Hash {
	var hashs []string
	for _, storagefile := range lease.StorageFile {
		hashs = append(hashs, storagefile.Hash.String())
	}
	for _, detail := range lease.LeaseList {
		hashs = append(hashs, detail.Hash.String())
	}
	hashs = append(hashs, lease.DepositAddress.String()+lease.UnitPrice.String()+lease.Capacity.String()+lease.RootHash.String()+lease.Address.String()+lease.Deposit.String()+strconv.Itoa(lease.Status)+lease.Cost.String()+
		lease.ValidationFailureTotalTime.String()+lease.LastVerificationSuccessTime.String()+lease.LastVerificationTime.String()+lease.Duration.String())
	sort.Strings(hashs)
	lease.Hash = getHash(hashs)
	s.accumulatePledgeHash(pledgeAddr) //accumulate  valid hash of Pledge
	return lease.Hash
}

/**
 *
 */
func (s *StorageData) accumulateSpaceHash(pledgeAddr common.Address) common.Hash {
	storageSpaces := s.StoragePledge[pledgeAddr].StorageSpaces
	var hashs []string
	for _, storagefile := range storageSpaces.StorageFile {
		hashs = append(hashs, storagefile.Hash.String())
	}
	hashs = append(hashs, storageSpaces.ValidationFailureTotalTime.String()+storageSpaces.LastVerificationSuccessTime.String()+storageSpaces.LastVerificationTime.String()+
		storageSpaces.Address.String()+storageSpaces.RootHash.String()+storageSpaces.StorageCapacity.String())
	sort.Strings(hashs)
	storageSpaces.Hash = getHash(hashs)
	s.accumulatePledgeHash(pledgeAddr) //accumulate  valid hash of Pledge
	return storageSpaces.Hash
}
func (s *StorageData) accumulatePledgeHash(pledgeAddr common.Address) common.Hash {
	storagePledge := s.StoragePledge[pledgeAddr]
	var hashs []string
	for _, lease := range storagePledge.Lease {
		hashs = append(hashs, lease.Hash.String())
	}
	hashs = append(hashs, storagePledge.Address.String()+
		storagePledge.LastVerificationTime.String()+
		storagePledge.LastVerificationSuccessTime.String()+
		storagePledge.ValidationFailureTotalTime.String()+
		storagePledge.Bandwidth.String()+
		storagePledge.PledgeStatus.String()+
		storagePledge.Number.String()+
		storagePledge.SpaceDeposit.String()+
		storagePledge.StorageSpaces.Hash.String()+
		storagePledge.Price.String()+
		storagePledge.StorageSize.String()+
		storagePledge.TotalCapacity.String())
	sort.Strings(hashs)
	storagePledge.Hash = getHash(hashs)
	return storagePledge.Hash
}

/**
*    accumulate   Validhash  of root hash
 */
func (s *StorageData) accumulateHeaderHash() common.Hash {
	var hashs []string
	for address, storagePledge := range s.StoragePledge {
		hashs = append(hashs, storagePledge.Hash.String(), address.Hash().String())
	}
	sort.Strings(hashs)
	s.Hash = getHash(hashs)
	return s.Hash
}

func getHash(obj interface{}) common.Hash {
	hasher := sha3.NewLegacyKeccak256()
	rlp.Encode(hasher, obj)
	var hash common.Hash
	hasher.Sum(hash[:0])
	return hash
}

func (s *StorageData) storageVerify(number uint64, blockPerday uint64, revenueStorage map[common.Address]*RevenueParameter) ([]common.Address, []common.Hash, map[common.Address]*StorageRatio) {
	sussSPAddrs := make([]common.Address, 0)
	sussRentHashs := make([]common.Hash, 0)
	storageRatios := make(map[common.Address]*StorageRatio, 0)

	bigNumber := new(big.Int).SetUint64(number)
	bigblockPerDay := new(big.Int).SetUint64(blockPerday)
	zeroTime := new(big.Int).Mul(new(big.Int).Div(bigNumber, bigblockPerDay), bigblockPerDay) //0:00 every day
	beforeZeroTime := new(big.Int).Sub(zeroTime, bigblockPerDay)
	bigOne := big.NewInt(1)
	for pledgeAddr, sPledge := range s.StoragePledge {
		isSfVerSucc := true
		capSucc := big.NewInt(0)
		rentSuccCount := 0
		storagespaces := s.StoragePledge[pledgeAddr].StorageSpaces
		sfiles := storagespaces.StorageFile
		for _, sfile := range sfiles {
			lastVerSuccTime := sfile.LastVerificationSuccessTime
			if lastVerSuccTime.Cmp(beforeZeroTime) < 0 {
				isSfVerSucc = false
				sfile.ValidationFailureTotalTime = new(big.Int).Add(sfile.ValidationFailureTotalTime, bigOne)
				s.accumulateSpaceStorageFileHash(pledgeAddr, sfile)
			} else {
				capSucc = new(big.Int).Add(capSucc, sfile.Capacity)
			}
		}
		if isSfVerSucc {
			storagespaces.LastVerificationSuccessTime = beforeZeroTime
		} else {
			storagespaces.ValidationFailureTotalTime = new(big.Int).Add(storagespaces.ValidationFailureTotalTime, bigOne)
		}
		storagespaces.LastVerificationTime = beforeZeroTime
		s.accumulateSpaceHash(pledgeAddr)
		leases := make(map[common.Hash]*Lease)
		for lhash, l := range sPledge.Lease {
			if l.Status == LeaseNormal || l.Status == LeaseBreach {
				leases[lhash] = l
			}
		}
		for lhash, lease := range leases {
			isVerSucc := true
			storageFile := lease.StorageFile
			for _, file := range storageFile {
				lastVerSuccTime := file.LastVerificationSuccessTime
				if lastVerSuccTime.Cmp(beforeZeroTime) < 0 {
					isVerSucc = false
					file.ValidationFailureTotalTime = new(big.Int).Add(file.ValidationFailureTotalTime, bigOne)
					s.accumulateLeaseStorageFileHash(pledgeAddr, lhash, file)
				} else {
					capSucc = new(big.Int).Add(capSucc, file.Capacity)
				}
			}
			leaseLists := lease.LeaseList
			expireNumber := big.NewInt(0)
			for _, leaseDetail := range leaseLists {
				deposit := leaseDetail.Deposit
				if deposit.Cmp(big.NewInt(0)) > 0 {
					startTime := leaseDetail.StartTime
					duration := leaseDetail.Duration
					leaseDetailEndNumber := new(big.Int).Add(startTime, new(big.Int).Mul(duration, new(big.Int).SetUint64(blockPerday)))
					if startTime.Cmp(beforeZeroTime) <= 0 && leaseDetailEndNumber.Cmp(beforeZeroTime) >= 0 {
						if !isVerSucc {
							leaseDetail.ValidationFailureTotalTime = new(big.Int).Add(lease.ValidationFailureTotalTime, bigOne)
							s.accumulateLeaseDetailHash(pledgeAddr, lhash, leaseDetail)
						}
					}
					if expireNumber.Cmp(leaseDetailEndNumber) < 0 {
						expireNumber = leaseDetailEndNumber
					}
				}
			}
			if expireNumber.Cmp(beforeZeroTime) <= 0 {
				lease.Status = LeaseExpiration
			}
			//cal ROOT HASH

			if isVerSucc {
				lease.LastVerificationSuccessTime = beforeZeroTime
				sussRentHashs = append(sussRentHashs, lhash)
				rentSuccCount++
				if lease.Status == LeaseBreach {
					duration10 := new(big.Int).Mul(lease.Duration, big.NewInt(rentFailToRescind))
					duration10 = new(big.Int).Div(duration10, big.NewInt(100))
					if lease.ValidationFailureTotalTime.Cmp(duration10) < 0 {
						lease.Status = LeaseNormal
					}
				}
			} else {
				lease.ValidationFailureTotalTime = new(big.Int).Add(lease.ValidationFailureTotalTime, bigOne)
				if lease.Status == LeaseNormal {
					duration10 := new(big.Int).Mul(lease.Duration, big.NewInt(rentFailToRescind))
					duration10 = new(big.Int).Div(duration10, big.NewInt(100))
					if lease.ValidationFailureTotalTime.Cmp(duration10) > 0 {
						lease.Status = LeaseBreach
					}
				}
			}
			lease.LastVerificationTime = beforeZeroTime
			s.accumulateLeaseHash(pledgeAddr, lease)
		}
		storageCapacity := storagespaces.StorageCapacity
		rent51 := len(leases) * 51 / 100
		isPledgeVerSucc := false
		cap90 := new(big.Int).Mul(big.NewInt(90), sPledge.TotalCapacity)
		cap90 = new(big.Int).Div(cap90, big.NewInt(100))
		if len(leases) == 0 {
			if capSucc.Cmp(cap90) >= 0 {
				isPledgeVerSucc = true
			}
		} else if storageCapacity.Cmp(big.NewInt(0)) == 0 {
			if rentSuccCount >= rent51 {
				isPledgeVerSucc = true
			}
		} else {
			if rentSuccCount >= rent51 && capSucc.Cmp(cap90) >= 0 {
				isPledgeVerSucc = true
			}
		}
		if isPledgeVerSucc {
			sussSPAddrs = append(sussSPAddrs, pledgeAddr)
			if revenue, ok := revenueStorage[pledgeAddr]; ok {
				if _, ok2 := storageRatios[revenue.RevenueAddress]; !ok2 {
					storageRatios[revenue.RevenueAddress] = &StorageRatio{
						Capacity: sPledge.TotalCapacity,
						Ratio:    decimal.NewFromInt(0),
					}
				} else {
					storageRatios[revenue.RevenueAddress].Capacity = new(big.Int).Add(storageRatios[revenue.RevenueAddress].Capacity, sPledge.TotalCapacity)
				}
			}
			sPledge.LastVerificationSuccessTime = beforeZeroTime
		} else {
			sPledge.ValidationFailureTotalTime = new(big.Int).Add(sPledge.ValidationFailureTotalTime, bigOne)
			maxFailNum := maxStgVerContinueDayFail * blockPerday
			bigMaxFailNum := new(big.Int).SetUint64(maxFailNum)
			if beforeZeroTime.Cmp(bigMaxFailNum) >= 0 {
				beforeSevenDayNumber := new(big.Int).Sub(beforeZeroTime, bigMaxFailNum)
				lastVerSuccTime := sPledge.LastVerificationSuccessTime
				if lastVerSuccTime.Cmp(beforeSevenDayNumber) < 0 {
					sPledge.PledgeStatus = big.NewInt(SPledgeRemoving)
				}
			}
		}
		sPledge.LastVerificationTime = beforeZeroTime
		s.accumulateSpaceHash(pledgeAddr)
	}
	//cal ROOT HASH
	s.accumulateHeaderHash()
	return sussSPAddrs, sussRentHashs, storageRatios
}

func (s *StorageData) dealLeaseStatus(number uint64, perday uint64, rate uint32, blockPerday uint64) ([]SpaceRewardRecord, []ExchangeSRTRecord) {
	revertLockReward := make([]SpaceRewardRecord, 0)
	revertExchangeSRT := make([]ExchangeSRTRecord, 0)
	delPledge := make([]common.Address, 0)
	for pledgeAddress, sPledge := range s.StoragePledge {
		if sPledge.PledgeStatus.Cmp(big.NewInt(SPledgeRetrun)) == 0 {
			continue
		}
		if sPledge.PledgeStatus.Cmp(big.NewInt(SPledgeRemoving)) == 0 || sPledge.PledgeStatus.Cmp(big.NewInt(SPledgeExit)) == 0 {
			sPledge.PledgeStatus = big.NewInt(SPledgeRetrun)
			revertLockReward, revertExchangeSRT = s.dealSPledgeRevert(revertLockReward, revertExchangeSRT, sPledge, rate, number, blockPerday)
			delPledge = append(delPledge, pledgeAddress)
			s.accumulateSpaceHash(pledgeAddress)
			continue
		}

		leases := sPledge.Lease
		for _, lease := range leases {
			if lease.Status == LeaseReturn {
				continue
			}
			if lease.Status == LeaseUserRescind || lease.Status == LeaseExpiration {
				lease.Status = LeaseReturn
				revertLockReward, revertExchangeSRT = s.dealLeaseRevert(lease, revertLockReward, revertExchangeSRT, rate)
				s.accumulateLeaseHash(pledgeAddress, lease)
			}
		}
	}
	for _, delAddr := range delPledge {
		delete(s.StoragePledge, delAddr)
	}
	s.accumulateHeaderHash()
	return revertLockReward, revertExchangeSRT
}

func (s *StorageData) dealSPledgeRevert(revertLockReward []SpaceRewardRecord, revertExchangeSRT []ExchangeSRTRecord, pledge *SPledge, rate uint32, number uint64, blockPerday uint64) ([]SpaceRewardRecord, []ExchangeSRTRecord) {
	revertLockReward, revertExchangeSRT = s.dealSPledgeRevert2(pledge, revertLockReward, revertExchangeSRT, rate,  number, blockPerday)
	leases := pledge.Lease
	for _, l := range leases {
		if l.Status == LeaseReturn {
			continue
		}
		revertLockReward, revertExchangeSRT = s.dealLeaseRevert(l, revertLockReward, revertExchangeSRT, rate)
	}
	return revertLockReward, revertExchangeSRT
}
func (s *StorageData) dealSPledgeRevert2(pledge *SPledge, revertLockReward []SpaceRewardRecord, revertExchangeSRT []ExchangeSRTRecord, rate uint32, number uint64, blockPerday uint64) ([]SpaceRewardRecord, []ExchangeSRTRecord) {
	bigNumber := new(big.Int).SetUint64(number)
	bigblockPerDay := new(big.Int).SetUint64(blockPerday)
	zeroTime := new(big.Int).Mul(new(big.Int).Div(bigNumber, bigblockPerDay), bigblockPerDay)
	startNumber := pledge.Number
	duration := new(big.Int).Sub(zeroTime, startNumber)
	duration = new(big.Int).Div(duration, bigblockPerDay)
	zero := big.NewInt(0)
	vFTT := pledge.ValidationFailureTotalTime
	deposit := pledge.SpaceDeposit
	depositAddress := pledge.Address
	revertDeposit := big.NewInt(0)
	if vFTT.Cmp(zero) > 0 {
		if duration.Cmp(vFTT) > 0 {
			revertAmount := new(big.Int).Mul(deposit, vFTT)
			revertAmount = new(big.Int).Div(revertAmount, duration)
			revertDeposit = new(big.Int).Sub(deposit, revertAmount)
		}
	} else {
		revertDeposit = deposit
	}
	if revertDeposit.Cmp(zero) > 0 {
		revertLockReward = append(revertLockReward, SpaceRewardRecord{
			Target:  depositAddress,
			Amount:  revertDeposit,
			Revenue: depositAddress,
		})
	}
	return revertLockReward, revertExchangeSRT
}

func (s *StorageData) dealLeaseRevert(l *Lease, revertLockReward []SpaceRewardRecord, revertExchangeSRT []ExchangeSRTRecord, rate uint32) ([]SpaceRewardRecord, []ExchangeSRTRecord) {
	zero := big.NewInt(0)
	vFTT := l.ValidationFailureTotalTime
	deposit := l.Deposit
	duration := l.Duration
	address := l.Address
	depositAddress := l.DepositAddress
	revertSRTAmount := big.NewInt(0)
	revertDeposit := big.NewInt(0)
	if vFTT.Cmp(zero) > 0 {
		if duration.Cmp(vFTT) > 0 {
			revertAmount := new(big.Int).Mul(deposit, vFTT)
			revertAmount = new(big.Int).Div(revertAmount, duration)
			revertDeposit = new(big.Int).Sub(deposit, revertAmount)
			revertAmount = new(big.Int).Div(new(big.Int).Mul(revertAmount, big.NewInt(int64(rate))), big.NewInt(10000))
			revertSRTAmount = new(big.Int).Add(revertSRTAmount, revertAmount)
		} else {
			revertAmount := new(big.Int).Div(new(big.Int).Mul(deposit, big.NewInt(int64(rate))), big.NewInt(10000))
			revertSRTAmount = new(big.Int).Add(revertSRTAmount, revertAmount)
		}
	} else {
		revertDeposit = deposit
	}
	if revertDeposit.Cmp(zero) > 0 {
		revertLockReward = append(revertLockReward, SpaceRewardRecord{
			Target:  depositAddress,
			Amount:  revertDeposit,
			Revenue: depositAddress,
		})

	}

	if revertSRTAmount.Cmp(zero) > 0 {
		revertExchangeSRT = append(revertExchangeSRT, ExchangeSRTRecord{
			Target: address,
			Amount: revertSRTAmount,
		})
	}
	return revertLockReward, revertExchangeSRT
}

func (s *StorageData) calcStorageRatio(ratios map[common.Address]*StorageRatio) map[common.Address]*StorageRatio {
	for _, ratio := range ratios {
		ratio.Ratio = s.calStorageRatio(ratio.Capacity)
	}
	return ratios
}

func (s *StorageData) calStorageRatio(totalCapacity *big.Int) decimal.Decimal {

	tb1b1024 := new(big.Int).Mul(big.NewInt(1024), tb1b)
	tb1b500 := new(big.Int).Mul(big.NewInt(500), tb1b)
	tb1b50 := new(big.Int).Mul(big.NewInt(50), tb1b)
	pd50 := new(big.Int).Mul(big.NewInt(50), tb1b1024)
	pd500 := new(big.Int).Mul(big.NewInt(500), tb1b1024)
	pd1024 := new(big.Int).Mul(big.NewInt(1024), tb1b1024)
	if totalCapacity.Cmp(pd1024) > 0 {
		return decimal.NewFromInt(2)
	}
	if totalCapacity.Cmp(pd1024) < 0 && totalCapacity.Cmp(pd500) > 0 {
		return decimal.NewFromFloat(1.8)
	}

	if totalCapacity.Cmp(pd500) <= 0 && totalCapacity.Cmp(pd50) > 0 {
		return decimal.NewFromFloat(1.5)
	}

	if totalCapacity.Cmp(pd50) <= 0 && totalCapacity.Cmp(tb1b1024) > 0 {
		return decimal.NewFromFloat(1.2)
	}
	if totalCapacity.Cmp(tb1b1024) == 0 {
		return decimal.NewFromInt(1)
	}
	if totalCapacity.Cmp(tb1b1024) < 0 && totalCapacity.Cmp(tb1b500) > 0 {
		return decimal.NewFromFloat(0.7)
	}
	if totalCapacity.Cmp(tb1b500) <= 0 && totalCapacity.Cmp(tb1b50) > 0 {
		return decimal.NewFromFloat(0.5)
	}
	if totalCapacity.Cmp(tb1b50) <= 0 && totalCapacity.Cmp(tb1b) > 0 {
		return decimal.NewFromFloat(0.3)
	}
	if totalCapacity.Cmp(tb1b) == 0 {
		return decimal.NewFromFloat(0.1)
	}
	return decimal.NewFromInt(0)
}

func (s *StorageData) calcStoragePledgeReward(ratios map[common.Address]*StorageRatio, revenueStorage map[common.Address]*RevenueParameter, number uint64, period uint64) ([]SpaceRewardRecord, *big.Int) {
	reward := make([]SpaceRewardRecord, 0)
	storageHarvest := big.NewInt(0)
	if nil == ratios || len(ratios) == 0 {
		return reward, storageHarvest
	}
	totalPledgeReward := big.NewInt(0)
	for pledgeAddr, sPledge := range s.StoragePledge {
		if revenue, ok := revenueStorage[pledgeAddr]; ok {
			if ratio, ok2 := ratios[revenue.RevenueAddress]; ok2 {
				bandwidthIndex := getBandwaith(sPledge.Bandwidth)
				pledgeReward := decimal.NewFromBigInt(sPledge.TotalCapacity, 0).Mul(bandwidthIndex).BigInt()
				pledgeReward = decimal.NewFromBigInt(pledgeReward, 0).Mul(ratio.Ratio).BigInt()
				totalPledgeReward = new(big.Int).Add(totalPledgeReward, pledgeReward)
			}
		}
	}
	if totalPledgeReward.Cmp(common.Big0) == 0 {
		return reward, storageHarvest
	}
	blockNumPerYear := secondsPerYear / period
	yearCount := number / blockNumPerYear

	var yearReward decimal.Decimal
	yearCount++
	if yearCount == 1 {
		yearReward = s.nYearSpaceProfitReward(float64(yearCount))
	} else {
		yearReward = s.nYearSpaceProfitReward(float64(yearCount)).Sub(s.nYearSpaceProfitReward(float64(yearCount - 1)))
	}
	spaceProfitReward := yearReward.Div(decimal.NewFromInt(365))

	for pledgeAddr, sPledge := range s.StoragePledge {
		if revenue, ok := revenueStorage[pledgeAddr]; ok {
			if ratio, ok2 := ratios[revenue.RevenueAddress]; ok2 {
				bandwidthIndex := getBandwaith(sPledge.Bandwidth)
				pledgeReward := decimal.NewFromBigInt(sPledge.TotalCapacity, 0).Mul(bandwidthIndex).BigInt()
				pledgeReward = decimal.NewFromBigInt(pledgeReward, 0).Mul(ratio.Ratio).BigInt()
				pledgeReward = decimal.NewFromBigInt(pledgeReward, 0).Mul(spaceProfitReward).BigInt()
				pledgeReward = new(big.Int).Div(pledgeReward, totalPledgeReward)
				reward = append(reward, SpaceRewardRecord{
					Target:  pledgeAddr,
					Amount:  pledgeReward,
					Revenue: revenue.RevenueAddress,
				})
				storageHarvest = new(big.Int).Add(storageHarvest, pledgeReward)


			}
		}
	}
	return reward, storageHarvest
}

func (s *StorageData) saveSpaceLockRewardTodb(reward []SpaceRewardRecord, storage map[common.Address]*RevenueParameter, db ethdb.Database, number uint64) error {
	key := fmt.Sprintf(storagePledgeRewardkey, number)
	blob, err := json.Marshal(reward)
	log.Info("saveSpaceLockRewardTodb", "key", key, "number", number, "reward", reward, "err", err)
	if err != nil {
		return err
	}

	err = db.Put([]byte(key), blob)
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageData) loadLockReward(db ethdb.Database, number uint64, rewardKey string) ([]SpaceRewardRecord, error) {
	key := fmt.Sprintf(rewardKey, number)
	blob, err := db.Get([]byte(key))
	if err != nil {
		log.Info("loadLockReward Get", "err", err)
		return nil, err
	}
	reward := make([]SpaceRewardRecord, 0)
	if err := json.Unmarshal(blob, &reward); err != nil {
		log.Info("loadLockReward Unmarshal", "err", err)
		return nil, err
	}
	return reward, nil
}

func (s *StorageData) saveLeaseLockRewardTodb(reward []SpaceRewardRecord, db ethdb.Database, number uint64) error {
	key := fmt.Sprintf(storageLeaseRewardkey, number)
	blob, err := json.Marshal(reward)
	if err != nil {
		return err
	}
	err = db.Put([]byte(key), blob)
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageData) deletePasstimeLease(number uint64, blockPerday uint64, passTime *big.Int) {
	bigNumber := new(big.Int).SetUint64(number)
	bigblockPerDay := new(big.Int).SetUint64(blockPerday)
	zeroTime := new(big.Int).Mul(new(big.Int).Div(bigNumber, bigblockPerDay), bigblockPerDay) //0:00 every day
	for pledgeAddr, sPledge := range s.StoragePledge {
		leases := sPledge.Lease
		delLeases := make([]common.Hash, 0)
		for h, lease := range leases {
			leaseDetails := lease.LeaseList
			delLeaseDetails := make([]common.Hash, 0)
			for hash, detail := range leaseDetails {
				deposit := detail.Deposit
				if deposit.Cmp(big.NewInt(0)) <= 0 {
					requestTime := detail.RequestTime
					requestTimeAddPassTime := new(big.Int).Add(requestTime, passTime)
					if requestTimeAddPassTime.Cmp(zeroTime) < 0 {
						delLeaseDetails = append(delLeaseDetails, hash)
					}
				}
			}
			for _, hash := range delLeaseDetails {
				delete(leaseDetails, hash)
				s.accumulateLeaseHash(pledgeAddr, lease)
			}
			if len(leaseDetails) == 0 {
				delLeases = append(delLeases, h)
			}
		}
		for _, hash := range delLeases {
			delete(leases, hash)
			s.accumulatePledgeHash(pledgeAddr)
		}
	}
	s.accumulateHeaderHash()
}

func (s *StorageData) saveSPledgeSuccTodb(addrs []common.Address, db ethdb.Database, number uint64) error {
	key := fmt.Sprintf("storagePleage-%d", number)
	blob, err := json.Marshal(addrs)
	if err != nil {
		return err
	}
	err = db.Put([]byte(key), blob)
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageData) saveRentSuccTodb(addrs []common.Hash, db ethdb.Database, number uint64) error {
	key := fmt.Sprintf("storageContract-%d", number)
	blob, err := json.Marshal(addrs)
	if err != nil {
		return err
	}
	err = db.Put([]byte(key), blob)
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageData) saveRevertSpaceLockRewardTodb(reward []SpaceRewardRecord, db ethdb.Database, number uint64) error {
	key := fmt.Sprintf(revertSpaceLockRewardkey, number)
	blob, err := json.Marshal(reward)
	if err != nil {
		return err
	}
	err = db.Put([]byte(key), blob)
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageData) saveRevertExchangeSRTTodb(exchangeSRT []ExchangeSRTRecord, db ethdb.Database, number uint64) error {
	key := fmt.Sprintf(revertExchangeSRTkey, number)
	blob, err := json.Marshal(exchangeSRT)
	if err != nil {
		return err
	}
	err = db.Put([]byte(key), blob)
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageData) lockStorageRatios(db ethdb.Database, number uint64) (map[common.Address]*StorageRatio, error) {
	key := fmt.Sprintf(storageRatioskey, number)
	blob, err := db.Get([]byte(key))
	if err != nil {
		log.Info("loadLockReward Get", "err", err)
		return nil, err
	}
	ratios := make(map[common.Address]*StorageRatio)
	if err := json.Unmarshal(blob, &ratios); err != nil {
		log.Info("loadLockReward Unmarshal", "err", err)
		return nil, err
	}
	return ratios, nil
}

func (s *StorageData) lockRevertSRT(db ethdb.Database, number uint64) ([]ExchangeSRTRecord, error) {
	key := fmt.Sprintf(revertExchangeSRTkey, number)
	blob, err := db.Get([]byte(key))
	if err != nil {
		log.Info("loadLockReward Get", "err", err)
		return nil, err
	}
	exchangeSRT := make([]ExchangeSRTRecord, 0)
	if err := json.Unmarshal(blob, &exchangeSRT); err != nil {
		log.Info("loadLockReward Unmarshal", "err", err)
		return nil, err
	}
	return exchangeSRT, nil
}

func (s *StorageData) saveStorageRatiosTodb(ratios map[common.Address]*StorageRatio, db ethdb.Database, number uint64) error {
	key := fmt.Sprintf(storageRatioskey, number)
	blob, err := json.Marshal(ratios)
	if err != nil {
		return err
	}
	err = db.Put([]byte(key), blob)
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageData) verifyParamsStoragePoc(txDataInfo []string, postion int, nonce uint64) (common.Hash, bool) {
	verifyData := txDataInfo[postion]
	verifyDataArr := strings.Split(verifyData, ",")
	RootHash := verifyDataArr[len(verifyDataArr)-1]
	if !verifyStoragePoc(verifyData, RootHash, nonce) {
		return common.Hash{}, false
	}
	return common.HexToHash(RootHash), true
}

func (s *Snapshot) updateExchangeSRT(exchangeSRT []ExchangeSRTRecord, headerNumber *big.Int, db ethdb.Database) {
	s.SRTIndex.updateExchangeSRT(exchangeSRT, headerNumber.Uint64(), db)
}

func (s *Snapshot) updateLeaseRequest(rent []LeaseRequestRecord, number *big.Int, db ethdb.Database) {
	if rent == nil || len(rent) == 0 {
		return
	}
	s.StorageData.updateLeaseRequest(rent, number, db)
}

func (s *Snapshot) updateLeasePledge(pg []LeasePledgeRecord, headerNumber *big.Int, db ethdb.Database) {
	if pg == nil || len(pg) == 0 {
		return
	}
	s.StorageData.updateLeasePledge(pg, headerNumber, db)
	s.SRTIndex.burnSRTAmount(pg, headerNumber.Uint64(), db)
}

func (s *Snapshot) updateLeaseRenewal(reNew []LeaseRenewalRecord, number *big.Int, db ethdb.Database) {
	if reNew == nil || len(reNew) == 0 {
		return
	}
	s.StorageData.updateLeaseRenewal(reNew, number, db, s.getBlockPreDay())
}

func (s *Snapshot) updateLeaseRenewalPledge(pg []LeaseRenewalPledgeRecord, headerNumber *big.Int, db ethdb.Database) {
	if pg == nil || len(pg) == 0 {
		return
	}
	s.StorageData.updateLeaseRenewalPledge(pg, headerNumber, db, s.getBlockPreDay())
	s.SRTIndex.burnSRTAmountReNew(pg, headerNumber.Uint64(), db)
}

func (s *Snapshot) updateLeaseRescind(rescinds []LeaseRescindRecord, number *big.Int, db ethdb.Database) {
	if rescinds == nil || len(rescinds) == 0 {
		return
	}
	s.StorageData.updateLeaseRescind(rescinds, number, db)
}

func (s *Snapshot) storageVerificationCheck(number uint64, blockPerday uint64, db ethdb.Database,currentLockReward [] LockRewardRecord) ([]LockRewardRecord,[]ExchangeSRTRecord, *big.Int,error) {
	if isStorageVerificationCheck(number, s.Period) {
		passTime := new(big.Int).Mul(s.SystemConfig.Deposit[sscEnumLeaseExpires], new(big.Int).SetUint64(blockPerday))
		basePrice := new(big.Int).Mul(s.SystemConfig.Deposit[sscEnumStoragePrice], big.NewInt(1024))
		return s.StorageData.storageVerificationCheck(number, blockPerday, passTime, s.SystemConfig.ExchRate, s.RevenueStorage, s.Period, db, basePrice, currentLockReward)
	}
	return currentLockReward,nil, nil,nil
}

func (snap *Snapshot) updateHarvest(harvest *big.Int) {
	if 0 < harvest.Cmp(big.NewInt(0)) {
		if nil == snap.FlowHarvest {
			snap.FlowHarvest = new(big.Int).Set(harvest)
		} else {
			snap.FlowHarvest = new(big.Int).Add(snap.FlowHarvest, harvest)
		}
	}
}

func (s *Snapshot) calStorageVerificationCheck(roothash common.Hash, number uint64, blockPerday uint64) (*Snapshot, error) {
	if isStorageVerificationCheck(number, s.Period) {
		passTime := new(big.Int).Mul(s.SystemConfig.Deposit[sscEnumLeaseExpires], new(big.Int).SetUint64(blockPerday))
		calRootHash := s.StorageData.calStorageVerificationCheck(number, blockPerday, passTime, s.RevenueStorage)
		if calRootHash != roothash {
			return s, errors.New("Storage root hash is not same,head:" + roothash.String() + "cal:" + calRootHash.String())
		}
	}
	return s, nil
}

func (s *StorageData) calStorageVerificationCheck(number uint64, blockPerday uint64, passTime *big.Int, revenueStorage map[common.Address]*RevenueParameter) common.Hash {
	s.storageVerify(number, blockPerday, revenueStorage)
	s.calDealLeaseStatus()
	s.deletePasstimeLease(number, blockPerday, passTime)
	return s.Hash
}

func (s *StorageData) calDealLeaseStatus() {
	delPledge := make([]common.Address, 0)
	for pledgeAddress, sPledge := range s.StoragePledge {
		if sPledge.PledgeStatus.Cmp(big.NewInt(SPledgeRetrun)) == 0 {
			continue
		}
		if sPledge.PledgeStatus.Cmp(big.NewInt(SPledgeRemoving)) == 0 || sPledge.PledgeStatus.Cmp(big.NewInt(SPledgeExit)) == 0 {
			sPledge.PledgeStatus = big.NewInt(SPledgeRetrun)
			delPledge = append(delPledge, pledgeAddress)
			s.accumulateSpaceHash(pledgeAddress)
			continue
		}

		leases := sPledge.Lease
		for _, lease := range leases {
			if lease.Status == LeaseReturn {
				continue
			}
			if lease.Status == LeaseUserRescind || lease.Status == LeaseExpiration {
				lease.Status = LeaseReturn
				s.accumulateLeaseHash(pledgeAddress, lease)
			}
		}
	}
	for _, delAddr := range delPledge {
		delete(s.StoragePledge, delAddr)
	}
	s.accumulateHeaderHash()
	return
}