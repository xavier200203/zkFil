package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"sync"

	pod_net "github.com/xuxinlai2002/zkFil/zkFil-node/net"
	"github.com/xuxinlai2002/zkFil/zkFil-node/net/rlpx"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// AliceStartNode starts p2p node for Alice node,
// listen to request from Bob
func AliceStartNode(AliceIPAddr string, key *keystore.Key, Log ILogger) error {

	var wg sync.WaitGroup

	serveAddr, err := rlpx.NewAddr(AliceIPAddr, key.PrivateKey.PublicKey)
	if err != nil {
		Log.Errorf("failed to initialize Alice's server address. err=%v", err)
		return fmt.Errorf("failed to start Alice's node")
	}
	// Log.Debugf("Initialize Alice's server address finish.")

	l, err := rlpx.Listen(serveAddr)
	if err != nil {
		Log.Errorf("failed to listen on %s: %v", serveAddr, err)
		return fmt.Errorf("failed to start Alice's node")
	}
	defer func() {
		if err := l.Close(); err != nil {
			Log.Errorf("failed to close listener on %s: %v",
				serveAddr, err)
		}
		if err := recover(); err != nil {
			Log.Errorf("exception unexpected: error=%v", err)
			return
		}
	}()
	AliceNodeStart = true
	fmt.Printf("===>>>Listen to %v\n\n", serveAddr)

	for {
		conn, err := l.Accept()
		if err != nil {
			Log.Errorf("failed to accept connection on %s: %v",
				serveAddr, err)
			continue
		}
		wg.Add(1)
		go func() {
			AliceAcceptTx(&wg, conn, key, Log)
			if err := conn.Close(); err != nil {
				Log.Errorf("failed to close connection on server side: %v",
					err)
				return
			}
		}()
	}
	wg.Wait()
	return nil
}

//AliceAcceptTx connects with Bob and handle transaction for Alice.
func AliceAcceptTx(wg *sync.WaitGroup, conn *rlpx.Connection, key *keystore.Key, Log ILogger) {
	Log.Infof("start connect with Bob node....")
	defer func() {
		wg.Done()
		if err := recover(); err != nil {
			Log.Errorf("exception unexpected: error=%v", err)
			return
		}
	}()

	node, rkey, params, err := preAliceTxAndConn(conn, key, Log)
	if err != nil {
		Log.Warnf("failed to prepare for transaction connection. err=%v", err)
		return
	}
	defer func() {
		if err := node.Close(); err != nil {
			Log.Errorf("failed to close server node: %v", err)
			return
		}
	}()
	Log.Debugf("[%v]step0: prepare for transaction successfully....", params.SessionID)

	var tx Transaction
	tx.SessionID = params.SessionID
	tx.Status = TRANSACTION_STATUS_START
	tx.BobPubKey = rkey
	tx.BobAddr = crypto.PubkeyToAddress(*rkey).Hex()
	tx.Bulletin = params.Bulletin
	tx.Mode = params.Mode
	tx.SubMode = params.SubMode
	tx.OT = params.OT
	tx.UnitPrice = params.UnitPrice
	tx.AliceAddr = key.Address.Hex()
	tx.Count = 1

	text := []uint8(tx.BobAddr)
	Log.Debugf("text:%v", text)

	AliceTxMap[tx.SessionID] = tx
	err = insertAliceTxToDB(tx)
	if err != nil {
		Log.Warnf("[%v]failed to save transaction to db for Alice. err=%v", params.SessionID, err)
		return
	}

	publishPath := BConf.AliceDir + "/publish/" + tx.Bulletin.SigmaMKLRoot

	if tx.Mode == TRANSACTION_MODE_PLAIN_POD {
		switch tx.SubMode {
		case TRANSACTION_SUB_MODE_COMPLAINT:
			if tx.OT {
				tx.PlainOTComplaint, err = AliceNewSessForPOC(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
				if err != nil {
					Log.Warnf("failed to prepare for Alice's session. err=%v", err)
					return
				}
				defer func() {
					tx.PlainOTComplaint.AliceSession.Free()
				}()
				Log.Debugf("success to prepare Alice session for plain_ot_complaint")

				err = AliceTxForPOC(node, key, tx, Log)
				if err != nil {
					Log.Warnf("transaction error. err=%v", err)
					return
				}
				Log.Debugf("transaction finish...")
			} else {
				tx.PlainComplaint, err = AliceNewSessForPC(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
				if err != nil {
					Log.Warnf("failed to prepare for Alice's session. err=%v", err)
					return
				}
				defer func() {
					tx.PlainComplaint.AliceSession.Free()
				}()
				Log.Debugf("success to prepare Alice session for plain_complaint")

				err = AliceTxForPC(node, key, tx, Log)
				if err != nil {
					Log.Warnf("transaction error. err=%v", err)
					return
				}
				Log.Debugf("transaction finish...")
			}
		case TRANSACTION_SUB_MODE_ATOMIC_SWAP:
			tx.PlainAtomicSwap, err = AliceNewSessForPAS(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
			if err != nil {
				Log.Warnf("Failed to prepare for Alice's session. err=%v", err)
				return
			}
			defer func() {
				tx.PlainAtomicSwap.AliceSession.Free()
			}()
			Log.Debugf("success to prepare Alice session for plain_atomic_swap")

			err = AliceTxForPAS(node, key, tx, Log)
			if err != nil {
				Log.Warnf("transaction error. err=%v", err)
				return
			}
			Log.Debugf("transaction finish...")
		case TRANSACTION_SUB_MODE_ATOMIC_SWAP_VC:
			tx.PlainAtomicSwapVc, err = AliceNewSessForPASVC(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
			if err != nil {
				Log.Warnf("failed to prepare for Alice's session. err=%v", err)
				return
			}
			defer func() {
				tx.PlainAtomicSwapVc.AliceSession.Free()
			}()
			Log.Debugf("success to prepare Alice session for plain_atomic_swap_vc")

			err = AliceTxForPASVC(node, key, tx, Log)
			if err != nil {
				Log.Warnf("transaction error. err=%v", err)
				return
			}
			Log.Debugf("transaction finish...")
		}
	} else if tx.Mode == TRANSACTION_MODE_TABLE_POD {
		switch tx.SubMode {
		case TRANSACTION_SUB_MODE_COMPLAINT:
			if tx.OT {
				tx.TableOTComplaint, err = AliceNewSessForTOC(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
				if err != nil {
					Log.Warnf("Failed to prepare for Alice's session. err=%v", err)
					return
				}
				defer func() {
					tx.TableOTComplaint.AliceSession.Free()
				}()
				Log.Debugf("success to prepare Alice session for table_ot_complaint1")

				err = AliceTxForTOC(node, key, tx, Log)
				if err != nil {
					Log.Warnf("transaction error. err=%v", err)
					return
				}
				Log.Debugf("transaction finish...")
			} else {
				tx.TableComplaint, err = AliceNewSessForTC(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
				if err != nil {
					Log.Warnf("Failed to prepare for Alice's session. err=%v", err)
					return
				}
				defer func() {
					tx.TableComplaint.AliceSession.Free()
				}()
				Log.Debugf("success to prepare Alice session for table_complaint")

				err = AliceTxForTC(node, key, tx, Log)
				if err != nil {
					Log.Warnf("transaction error. err=%v", err)
					return
				}
				Log.Debugf("transaction finish...")
			}
		case TRANSACTION_SUB_MODE_ATOMIC_SWAP:
			tx.TableAtomicSwap, err = AliceNewSessForTAS(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
			if err != nil {
				Log.Warnf("Failed to prepare for Alice's session. err=%v", err)
				return
			}
			defer func() {
				tx.TableAtomicSwap.AliceSession.Free()
			}()
			Log.Debugf("success to prepare Alice session for table_atomic_swap")

			err = AliceTxForTAS(node, key, tx, Log)
			if err != nil {
				Log.Warnf("transaction error. err=%v", err)
				return
			}
			Log.Debugf("transaction finish...")
		case TRANSACTION_SUB_MODE_ATOMIC_SWAP_VC:
			tx.TableAtomicSwapVc, err = AliceNewSessForTASVC(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
			if err != nil {
				Log.Warnf("failed to prepare for Alice's session. err=%v", err)
				return
			}
			defer func() {
				tx.TableAtomicSwapVc.AliceSession.Free()
			}()
			Log.Debugf("success to prepare Alice session for table_atomic_swap_vc")

			err = AliceTxForTASVC(node, key, tx, Log)
			if err != nil {
				Log.Warnf("transaction error. err=%v", err)
				return
			}
			Log.Debugf("transaction finish...")
		case TRANSACTION_SUB_MODE_VRF:
			if tx.OT {
				tx.TableOTVRF, err = AliceNewSessForTOQ(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
				if err != nil {
					Log.Warnf("Failed to prepare for Alice's session. err=%v", err)
					return
				}
				defer func() {
					tx.TableOTVRF.AliceSession.Free()
				}()
				Log.Debugf("success to prepare Alice session for table_ot_vrf")

				err = AliceTxForTOQ(node, key, tx, Log)
				if err != nil {
					Log.Warnf("transaction error. err=%v", err)
					return
				}
				Log.Debugf("transaction finish...")
			} else {
				tx.TableVRF, err = AliceNewSessForTQ(publishPath, converAddr(tx.AliceAddr), converAddr(tx.BobAddr), Log)
				if err != nil {
					Log.Warnf("Failed to prepare for Alice's session. err=%v", err)
					return
				}
				defer func() {
					tx.TableVRF.AliceSession.Free()
				}()
				Log.Debugf("success to prepare Alice session for table_vrf")

				err = AliceTxForTQ(node, key, tx, Log)
				if err != nil {
					Log.Warnf("transaction error. err=%v", err)
					return
				}
				Log.Debugf("transaction finish...")
			}
		}
	}
}

type AliceConnParam struct {
	Mode      string
	SubMode   string
	OT        bool
	UnitPrice int64
	SessionID string
	Bulletin  Bulletin
}

func preAliceTxAndConn(conn *rlpx.Connection, key *keystore.Key, Log ILogger) (node *pod_net.Node, rkey *ecdsa.PublicKey, params AliceConnParam, err error) {

	node, rkey, err = AliceNewConn(conn, key, Log)
	if err != nil {
		Log.Warnf("%v", err)
		return
	}
	Log.Debugf("success to new connection...")

	req, re, err := AliceRcvSessReq(node, Log)
	if err != nil {
		node.Close()
		Log.Warnf("failed to receive session request. err=%v", err)
		return
	}

	mklroot := hex.EncodeToString(req.SigmaMklRoot)
	params, re, err = preAliceTx(mklroot, re, Log)
	if err != nil {
		node.Close()
		Log.Warnf("failed to prepare for transaction. err=%v", err)
		err = fmt.Errorf("failed to prepare for transaction")
		return
	}
	Log.Debugf("[%v]success to prepare for transaction...", params.SessionID)

	req.ExtraInfo, err = json.Marshal(&re)
	if err != nil {
		node.Close()
		Log.Warnf("failed to marshal extra info. err=%v")
		err = fmt.Errorf("failed to save extra info")
		return
	}

	err = preAliceConn(node, key, params, req, Log)
	if err != nil {
		node.Close()
		Log.Errorf("failed to establish session with Bob. err=%v", err)
		return
	}
	Log.Debugf("[%v]established connection session successfully....", params.SessionID)
	return
}

func AliceNewConn(conn *rlpx.Connection, key *keystore.Key, Log ILogger) (*pod_net.Node, *ecdsa.PublicKey, error) {

	rkey, err := conn.Handshake(key.PrivateKey, false)
	if err != nil {
		Log.Errorf("failed to server-side handshake: %v", err)
		return nil, rkey, errors.New("failed to server-side handshake")
	}
	Log.Debugf("establish connection's handshake successfully...")

	node, err := pod_net.NewNode(conn, key.PrivateKey, rkey)
	if err != nil {
		Log.Errorf("failed to create server node: %v", err)
		return nil, rkey, errors.New("failed to create server node")
	}
	Log.Debugf("create connection node successfully....")
	return node, rkey, nil
}

type requestExtra struct {
	Price   int64  `json:"price"`
	Mode    string `json:"mode"`
	SubMode string `json:"subMode"`
	Ot      bool   `json:"ot"`
}

func AliceRcvSessReq(node *pod_net.Node, Log ILogger) (req *pod_net.SessionRequest, re requestExtra, err error) {
	req, err = node.RecvSessionRequest()
	if err != nil {
		Log.Warnf("failed to receive session request. err=%v", err)
		err = fmt.Errorf("failed to receive session request")
		return
	}
	if req.ID != 0 {
		Log.Warnf("session ID (%d) not zero", req.ID)
		err = fmt.Errorf("session ID not zero")
		return
	}
	Log.Debugf("success to receive session request...")
	err = json.Unmarshal(req.ExtraInfo, &re)
	if err != nil {
		Log.Warnf("failed to parse extra info. err=%v")
		err = fmt.Errorf("failed to parse extra info")
		return
	}
	return
}

func preAliceConn(node *pod_net.Node, key *keystore.Key, params AliceConnParam, req *pod_net.SessionRequest, Log ILogger) (err error) {

	/////////////////////////RecvSessionRequest/////////////////////////
	sessionIDInt, err := strconv.ParseUint(params.SessionID, 16, 64)
	if err != nil {
		Log.Warnf("failed to convert sessionID. err=%v", err)
		err = fmt.Errorf("failed to convert sessionID")
		return
	}

	netMode, err := modeToInt(params.Mode, params.SubMode, params.OT)
	if err != nil {
		Log.Warnf("failed to convert mode to netMode. mode=%v, subMode=%v, ot=%v", params.Mode, params.SubMode, params.OT)
		err = fmt.Errorf("failed to convert mode to netMode")
		return
	}

	/////////////////////////SendSessionAck/////////////////////////
	if err = node.SendSessionAck(
		sessionIDInt, netMode, req.SigmaMklRoot, req.ExtraInfo, true,
	); err != nil {
		err = fmt.Errorf(
			"failed to send session ack from server: %v",
			err)
		return
	}
	Log.Debugf("success to send session ack...")

	/////////////////////////RecvSessionAck/////////////////////////
	ack, err := node.RecvSessionAck(false)
	if err != nil {
		err = fmt.Errorf(
			"failed to receive session ack on server node: %v",
			err)
		return
	}
	if ack.ID != sessionIDInt {
		err = fmt.Errorf(
			"mismatch session ID on server node, get %d, expect %d",
			ack.ID, sessionIDInt)
		return
	}
	Log.Debugf("success to receive session ack...")
	return
}

func modeToInt(mode string, subMode string, ot bool) (netMode uint8, err error) {

	if mode == TRANSACTION_MODE_PLAIN_POD {
		switch subMode {
		case TRANSACTION_SUB_MODE_COMPLAINT:
			if !ot {
				netMode = pod_net.ModePlainComplaintPoD
			} else {
				netMode = pod_net.ModePlainOTComplaintPoD
			}
		case TRANSACTION_SUB_MODE_ATOMIC_SWAP:
			netMode = pod_net.ModePlainAtomicSwapPoD
		case TRANSACTION_SUB_MODE_ATOMIC_SWAP_VC:
			netMode = pod_net.ModePlainAtomicSwapVcPoD
		default:
			err = errors.New("invalid mode")
		}
	} else if mode == TRANSACTION_MODE_TABLE_POD {
		switch subMode {
		case TRANSACTION_SUB_MODE_COMPLAINT:
			if !ot {
				netMode = pod_net.ModeTableComplaintPoD
			} else {
				netMode = pod_net.ModeTableOTComplaintPoD
			}
		case TRANSACTION_SUB_MODE_ATOMIC_SWAP:
			netMode = pod_net.ModeTableAtomicSwapPoD
		case TRANSACTION_SUB_MODE_ATOMIC_SWAP_VC:
			netMode = pod_net.ModeTableAtomicSwapVcPoD
		case TRANSACTION_SUB_MODE_VRF:
			if !ot {
				netMode = pod_net.ModeTableVRFQuery
			} else {
				netMode = pod_net.ModeTableOTVRFQuery
			}
		default:
			err = errors.New("invalid mode")
		}
	} else {
		err = errors.New("invalid mode")
	}
	return
}

func modeFromInt(netMode uint8) (mode string, subMode string, ot bool, err error) {
	switch netMode {
	case pod_net.ModePlainComplaintPoD:
		mode = TRANSACTION_MODE_PLAIN_POD
		subMode = TRANSACTION_SUB_MODE_COMPLAINT
		ot = false
	case pod_net.ModePlainOTComplaintPoD:
		mode = TRANSACTION_MODE_PLAIN_POD
		subMode = TRANSACTION_SUB_MODE_COMPLAINT
		ot = true
	case pod_net.ModePlainAtomicSwapPoD:
		mode = TRANSACTION_MODE_PLAIN_POD
		subMode = TRANSACTION_SUB_MODE_ATOMIC_SWAP
		ot = false
	case pod_net.ModePlainAtomicSwapVcPoD:
		mode = TRANSACTION_MODE_PLAIN_POD
		subMode = TRANSACTION_SUB_MODE_ATOMIC_SWAP_VC
		ot = false
	case pod_net.ModeTableComplaintPoD:
		mode = TRANSACTION_MODE_TABLE_POD
		subMode = TRANSACTION_SUB_MODE_COMPLAINT
		ot = false
	case pod_net.ModeTableOTComplaintPoD:
		mode = TRANSACTION_MODE_TABLE_POD
		subMode = TRANSACTION_SUB_MODE_COMPLAINT
		ot = true
	case pod_net.ModeTableAtomicSwapPoD:
		mode = TRANSACTION_MODE_TABLE_POD
		subMode = TRANSACTION_SUB_MODE_ATOMIC_SWAP
		ot = false
	case pod_net.ModeTableAtomicSwapVcPoD:
		mode = TRANSACTION_MODE_TABLE_POD
		subMode = TRANSACTION_SUB_MODE_ATOMIC_SWAP_VC
		ot = false
	case pod_net.ModeTableVRFQuery:
		mode = TRANSACTION_MODE_TABLE_POD
		subMode = TRANSACTION_SUB_MODE_VRF
		ot = false
	case pod_net.ModeTableOTVRFQuery:
		mode = TRANSACTION_MODE_TABLE_POD
		subMode = TRANSACTION_SUB_MODE_VRF
		ot = true
	default:
		err = fmt.Errorf("invalid mode=%v", netMode)
	}
	return
}

func AliceRcvPODReq(node *pod_net.Node, requestFile string) error {
	reqBuf := new(bytes.Buffer)
	if _, err := node.RecvTxRequest(reqBuf); err != nil {
		return err
	}

	reqf, err := os.OpenFile(requestFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}
	defer reqf.Close()

	_, err = reqf.Write(reqBuf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write to request to file: %v", err)
	}
	return nil
}

func AliceSendPODResp(node *pod_net.Node, responseFile string) error {
	txResponse, err := ioutil.ReadFile(responseFile)
	if err != nil {
		return fmt.Errorf("failed to read response file: %v", err)
	}
	if err := node.SendTxResponse(
		bytes.NewReader(txResponse), uint64(len(txResponse)),
	); err != nil {
		return err
	}
	return nil
}

func AliceRcvPODRecpt(node *pod_net.Node, receiptFile string) (receiptSign []byte, price int64, expireAt int64, err error) {
	receipt, _, err := node.RecvTxReceipt()
	if err != nil {
		return
	}
	var receiptConn ReceiptForConnection
	err = json.Unmarshal(receipt, &receiptConn)
	if err != nil {
		err = fmt.Errorf("failed to parse receipt. err=%v", err)
		return
	}
	price = receiptConn.Price
	expireAt = receiptConn.ExpireAt
	receiptSign = receiptConn.ReceiptSign

	recf, err := os.OpenFile(receiptFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		err = fmt.Errorf("failed to save file: %v", err)
		return
	}
	defer recf.Close()

	_, err = recf.Write(receiptConn.ReceiptByte)
	if err != nil {
		err = fmt.Errorf("failed to write to receipt to file: %v", err)
		return
	}
	return
}

func AliceReceiveNegoReq(node *pod_net.Node, BobNegoRequestFile string) error {
	reqBuf := new(bytes.Buffer)
	if _, err := node.RecvNegoRequest(reqBuf); err != nil {
		return fmt.Errorf(
			"failed to receive negotiation request: %v",
			err)
	}
	reqf, err := os.OpenFile(BobNegoRequestFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}
	defer reqf.Close()

	_, err = reqf.Write(reqBuf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write to nego request to file: %v", err)
	}
	return nil
}

func AliceSendNegoResp(node *pod_net.Node, negoResponseFile string, negoRequestFile string) error {
	txResponse, err := ioutil.ReadFile(negoResponseFile)
	if err != nil {
		return fmt.Errorf("failed to read response file: %v", err)
	}
	txRequest, err := ioutil.ReadFile(negoRequestFile)
	if err != nil {
		return fmt.Errorf("failed to read ack file: %v", err)
	}

	if err := node.SendNegoAckReq(
		bytes.NewReader(txResponse),
		bytes.NewReader(txRequest),
		uint64(len(txResponse)),
		uint64(len(txRequest)),
	); err != nil {
		return fmt.Errorf(
			"failed to send nego ack+req: %v", err)
	}
	return nil
}

func AliceRcvNegoResp(node *pod_net.Node, BobNegoResponseFile string) error {

	negoRespBuf := new(bytes.Buffer)
	if _, err := node.RecvNegoAck(negoRespBuf); err != nil {
		return fmt.Errorf(
			"failed to receive nego ack: %v", err)
	}

	reqf, err := os.OpenFile(BobNegoResponseFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}
	defer reqf.Close()

	_, err = reqf.Write(negoRespBuf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write to nego request to file: %v", err)
	}
	return nil
}

/////////////////////////////////Bob////////////////////////////////////////
type BobConnParam struct {
	AliceIPAddr string
	AliceAddr   string
	Mode        string
	SubMode     string
	OT          bool
	UnitPrice   int64
	SessionID   string
	MerkleRoot  string
}

func preBobConn(params BobConnParam, key *keystore.Key, Log ILogger) (*pod_net.Node, *rlpx.Connection, BobConnParam, error) {
	node, conn, err := buyNewConn(params.AliceIPAddr, params.AliceAddr, key, Log)
	if err != nil {
		Log.Warnf("failed to new connection. err=%v", err)
		return node, conn, params, errors.New("failed to new connection")
	}
	params.SessionID, params.Mode, params.SubMode, params.OT, err = BobCreateSess(node, params.MerkleRoot, params.Mode, params.SubMode, params.OT, params.UnitPrice, Log)
	if err != nil {
		Log.Warnf("failed to create net session. err=%v", err)
		return node, conn, params, errors.New("failed to create net session")
	}
	dir := BConf.BobDir + "/transaction/" + params.SessionID
	err = os.Mkdir(dir, os.ModePerm)
	if err != nil {
		Log.Errorf("create folder %v error. err=%v", dir, err)
		return node, conn, params, errors.New("failed to create folder")
	}
	Log.Debugf("success to create folder. dir=%v", dir)
	return node, conn, params, nil
}

func buyNewConn(AliceIPAddr string, AliceAddr string, key *keystore.Key, Log ILogger) (*pod_net.Node, *rlpx.Connection, error) {

	Log.Debugf("AliceIPAddr=%v", AliceIPAddr)
	Log.Debugf("PublicKey=%v", key.PrivateKey.PublicKey)
	commonAddr := common.HexToAddress(AliceAddr)
	tcpAddr, err := net.ResolveTCPAddr("tcp", AliceIPAddr)
	if err != nil {
		return nil, nil, err
	}
	serveAddr := &rlpx.Addr{
		TCPAddr: tcpAddr,
		EthAddr: commonAddr,
	}
	Log.Debugf("serveAddr=%v", serveAddr)

	conn, err := rlpx.Dial(serveAddr)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to dial %s: %v", AliceIPAddr, err)
	}
	Log.Debugf("create dial with Alice successfully. AliceAddr=%v, AliceIP=%v. ", serveAddr, AliceIPAddr)

	rkey, err := conn.Handshake(key.PrivateKey, true)
	if err != nil {
		return nil, nil, fmt.Errorf("client-side handshake failed: %v", err)
	}
	Log.Debugf("establish connection handshake successfully...")

	node, err := pod_net.NewNode(conn, key.PrivateKey, rkey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client node: %v", err)
	}
	Log.Debugf("connect to Alice...")
	return node, conn, nil
}

func BobCreateSess(node *pod_net.Node, mklroot string, mode string, subMode string, ot bool, unitPrice int64, Log ILogger) (string, string, string, bool, error) {
	mklrootByte, err := hex.DecodeString(mklroot)
	if err != nil {
		return "", mode, subMode, ot, fmt.Errorf(
			"failed to decode merkle root: %v",
			err)
	}

	var extra requestExtra
	extra.Price = unitPrice
	extra.Mode = mode
	extra.Ot = ot
	extra.SubMode = subMode
	extraByte, err := json.Marshal(&extra)
	if err != nil {
		return "", mode, subMode, ot, fmt.Errorf(
			"failed to decode extra info: %v",
			err)
	}
	Log.Debugf("extra = %v", string(extraByte))

	/////////////////////////SendNewSessionRequest/////////////////////////
	if err = node.SendNewSessionRequest(
		uint8(0), mklrootByte, extraByte,
	); err != nil {
		return "", mode, subMode, ot, fmt.Errorf(
			"failed to send session request: %v",
			err)
	}
	Log.Debugf("success to send session request...")

	/////////////////////////RecvSessionAck/////////////////////////
	ack, err := node.RecvSessionAck(true)
	if err != nil {
		return "", mode, subMode, ot, fmt.Errorf(
			"failed to receive session ack on client node: %v",
			err)
	}
	Log.Debugf("success to receive session ack...%v", ack.ID)

	mode, subMode, ot, err = modeFromInt(ack.Mode)
	if err != nil {
		return "", mode, subMode, ot, fmt.Errorf(
			"invalid net mode: %v",
			ack.Mode)
	}

	/////////////////////////SendSessionAck/////////////////////////
	if err := node.SendSessionAck(
		ack.ID, ack.Mode, mklrootByte, ack.ExtraInfo, false,
	); err != nil {
		return "", mode, subMode, ot, fmt.Errorf(
			"failed to send session ack from client: %v",
			err)
	}
	Log.Debugf("success to send session ack...")

	sessionID := fmt.Sprintf("%x", ack.ID)
	return sessionID, mode, subMode, ot, nil
}

func BobSendPODReq(node *pod_net.Node, requestFile string) error {
	txReq, err := ioutil.ReadFile(requestFile)
	if err != nil {
		return fmt.Errorf("failed to read transaction request file: %v", err)
	}
	if err = node.SendTxRequest(bytes.NewReader(txReq), uint64(len(txReq))); err != nil {
		return fmt.Errorf(
			"failed to send Tx request: %v", err)
	}
	return nil
}

func BobRcvPODResp(node *pod_net.Node, responseFile string) error {

	RespBuf := new(bytes.Buffer)
	_, err := node.RecvTxResponse(RespBuf)
	if err != nil {
		return err
	}
	respf, err := os.OpenFile(responseFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to save file")
	}
	defer respf.Close()

	_, err = respf.Write(RespBuf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write to nego request to file: %v", err)
	}
	return nil
}

type ReceiptForConnection struct {
	ReceiptByte []byte `json:"receiptByte"`
	ReceiptSign []byte `json:"receiptSign"`
	Price       int64  `json:"price"`
	ExpireAt    int64  `json:"expireAt"`
}

func BobSendPODRecpt(node *pod_net.Node, price int64, expireAt int64, receiptByte []byte, sign []byte) error {

	var receipt ReceiptForConnection
	receipt.ReceiptByte = receiptByte
	receipt.ReceiptSign = sign
	receipt.Price = price
	receipt.ExpireAt = expireAt
	receiptConnBytes, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	err = node.SendTxReceipt(bytes.NewReader(receiptConnBytes), uint64(len(receiptConnBytes)))
	if err != nil {
		return err
	}
	return nil
}

func BobSendNegoReq(node *pod_net.Node, negoRequestFile string) error {

	BobNegoReq, err := ioutil.ReadFile(negoRequestFile)
	if err != nil {
		return fmt.Errorf("failed to read transaction receipt file: %v", err)
	}

	if err := node.SendNegoRequest(
		bytes.NewReader(BobNegoReq),
		uint64(len(BobNegoReq)),
	); err != nil {
		return fmt.Errorf(
			"failed to send negotiation request: %v",
			err)
	}

	return nil
}

func BobRcvNegoResp(node *pod_net.Node, negoResponseFile string, negoAckFile string) error {
	respBuf := new(bytes.Buffer)
	ackBuf := new(bytes.Buffer)
	if _, _, err := node.RecvNegoAckReq(
		respBuf, ackBuf,
	); err != nil {
		return fmt.Errorf(
			"failed to receive nego ack+req: %v",
			err)
	}

	respf, err := os.OpenFile(negoResponseFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to save file")
	}
	defer respf.Close()

	_, err = respf.Write(respBuf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write to nego request to file")
	}

	ackf, err := os.OpenFile(negoAckFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to save file")
	}
	defer ackf.Close()

	_, err = ackf.Write(ackBuf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write to nego request to file")
	}
	return nil
}

func BobSendNegoResp(node *pod_net.Node, negoResponseFile string) error {

	BobNegoResp, err := ioutil.ReadFile(negoResponseFile)
	if err != nil {
		return fmt.Errorf("failed to read transaction receipt file: %v", err)
	}

	if err := node.SendNegoAck(
		bytes.NewReader(BobNegoResp),
		uint64(len(BobNegoResp)),
	); err != nil {
		return fmt.Errorf(
			"failed to send nego ack: %v", err)
	}
	return nil
}
