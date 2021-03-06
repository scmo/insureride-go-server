package ethereum

import (
	"github.com/astaxie/beego"
	_ "github.com/lib/pq"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"strings"
	"github.com/ethereum/go-ethereum/common"
	"strconv"
	"math/big"
	"github.com/scmo/insureride-go-server/ethereum/smartcontract"
	"github.com/scmo/insureride-go-server/models"
)

type EthereumController struct {
	Auth   *bind.TransactOpts
	Client *ethclient.Client
}

var ethereumController EthereumController

func Init() {
	dataDir := beego.AppConfig.String("dataDir")
	if beego.BConfig.RunMode == "dev" {
		dataDir = dataDir + "testnet/"
	}

	ipcfile := dataDir + "geth.ipc"

	// Create an IPC based RPC connection to a remote node
	client, err := ethclient.Dial(ipcfile)
	if err != nil {
		beego.Critical("Failed to connect to the Ethereum client: ", err)
	}
	auth, err := bind.NewTransactor(strings.NewReader(beego.AppConfig.String("systemAccount")), beego.AppConfig.String("systemAccountPassword"))
	if err != nil {
		beego.Critical("Failed to create authorized transactor: ", err)
	}
	ethereumController = EthereumController{Auth:auth, Client:client}
}

func getContractCarSession(contractCar *smartcontract.ContractCar) (*smartcontract.ContractCarSession) {
	contractCarSession := &smartcontract.ContractCarSession{
		Contract: contractCar,
		CallOpts: bind.CallOpts{Pending:true},
		TransactOpts: bind.TransactOpts{
			From:ethereumController.Auth.From,
			Signer:ethereumController.Auth.Signer,
			GasLimit:big.NewInt(3141592),
		},
	}
	return contractCarSession
}

func getContractDriveSession(contractDrive *smartcontract.ContractDrive) (*smartcontract.ContractDriveSession) {
	contractDriveSession := &smartcontract.ContractDriveSession{
		Contract: contractDrive,
		CallOpts: bind.CallOpts{Pending:true},
		TransactOpts: bind.TransactOpts{
			From:ethereumController.Auth.From,
			Signer:ethereumController.Auth.Signer,
			GasLimit:big.NewInt(3141592),
		},
	}
	return contractDriveSession
}

func GetCar(car *models.Car) (*models.Car, error) {
	contractcar, err := smartcontract.NewContractCar(common.HexToAddress(car.ContractAddress), ethereumController.Client)
	if err != nil {
		beego.Critical("Failed to instantiate a Token contract: %v", err)
	}
	session := getContractCarSession(contractcar)
	car.Brand, err = session.Brand()
	car.Model, err = session.Model()
	car.Year, err = session.Year()
	car.Vehiclenumber, err = session.Vehiclenumber()
	car.BalanceInt, err = session.Balance()
	car.Balance = float32(car.BalanceInt) / 100

	drivescount, _ := session.Nodrives()

	bigstr := drivescount.String()
	numberofdrives, err := strconv.Atoi(bigstr)
	car.Drives = make([]*models.Drive, numberofdrives)
	for i := 0; i < numberofdrives; i++ {
		add, err := session.Drives(big.NewInt(int64(i)));
		if (err != nil) {
			beego.Critical("Failed getting arraz ", err)
		}
		drive := models.Drive{}
		drive.ContractAddress = add.String()
		GetDrive(&drive)
		car.Drives[i] = &drive
	}
	return car, err
}

func GetDrive(drive *models.Drive) (*models.Drive, error) {
	contractDrive, err := smartcontract.NewContractDrive(common.HexToAddress(drive.ContractAddress), ethereumController.Client)
	if err != nil {
		beego.Critical("Failed to instantiate a Token contract: %v", err)
	}
	session := getContractDriveSession(contractDrive)
	drive.Kilometers, _ = string2Float64(session.Kilometers())
	drive.Avgaccel, _ = string2Float64(session.Avgaccel())
	drive.Avgspeed, _ = string2Float64(session.Avgspeed())
	drive.PriceInt, _ = session.Price()
	drive.Starttime, _ = session.Starttime()
	drive.Endtime, _ = session.Endtime()
	drive.Price = float32(drive.PriceInt) / 100
	return drive, err
}


// Deploys a drive contract to the blockchain
func AddDrive(d models.Drive) (models.Drive, error) {
	address, tx, _, err := smartcontract.DeployContractDrive(ethereumController.Auth, ethereumController.Client, float64ToString(d.Kilometers), float64ToString(d.Avgspeed), float64ToString(d.Avgaccel), d.Starttime, d.Endtime, d.PriceInt)
	if err != nil {
		beego.Critical("Failed to deploy new token contract: ", err)
	}
	beego.Info("Contract pending deploy: ", address.String())
	beego.Info("Transaction waiting to be mined: ", tx.Hash().String())
	d.ContractAddress = address.String()
	return d, err
}

func AddDriveToCar(carContractAddress string, driveContractAddress string) {
	contractcar, err := smartcontract.NewContractCar(common.HexToAddress(carContractAddress), ethereumController.Client)
	if err != nil {
		beego.Critical("Failed to instantiate a Token contract: %v", err)
	}
	session := getContractCarSession(contractcar)

	tx, err := session.AddDrive(common.HexToAddress(driveContractAddress))
	if (err != nil) {
		beego.Critical(err)
	}
	beego.Info("Transaction waiting to be mined: ", tx.Hash().String())
}

func PayInsurance(carContractAddress string, amount uint16){
	contractcar, err := smartcontract.NewContractCar(common.HexToAddress(carContractAddress), ethereumController.Client)
	if err != nil {
		beego.Critical("Failed to instantiate a Token contract: %v", err)
	}
	session := getContractCarSession(contractcar)
	_ = session
	tx, err := session.PayInsurance(amount);
	if (err != nil) {
		beego.Critical(err)
	}
	beego.Info("Transaction waiting to be mined: ", tx.Hash().String())
}

func float64ToString(f float64) (string) {
	return strconv.FormatFloat(f, 'f', 6, 64)
}

func string2Float64(s string, err error) (float64, error) {
	f, err := strconv.ParseFloat(s, 64)
	if (err != nil) {
		beego.Error(err)
	}
	return f, err
}
