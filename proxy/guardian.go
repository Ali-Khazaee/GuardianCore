package proxy

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/xtls/xray-core/common/uuid"
)

type Account struct {
	Day     uint32
	Time    uint32
	Flag    string
	Traffic uint32
	Usage   uint32

	Trace   uint32
	Refresh uint32

	// UserMap map[string]uint32
}

var AccountMapMutex = sync.Mutex{}
var AccountMap = make(map[string]*Account)

func init() {
	if DBError != nil {
		panic(DBError)
	}

	DB.SetConnMaxLifetime(time.Minute * 10)
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(10)
}

func AccountVerifyVLESS(AccountUUID uuid.UUID, AccountIP net.Addr) bool {
	AccountMapMutex.Lock()
	defer AccountMapMutex.Unlock()

	var AccountKey = AccountUUID.String()

	Cache, OK := AccountMap[AccountKey]

	if !OK {
		Cache = new(Account)

		Error := DB.QueryRow("SELECT `day`, `time`, `flag`, `traffic`, `usage` FROM `subscription` WHERE `uuid` = ? LIMIT 1;", AccountKey).Scan(&Cache.Day, &Cache.Time, &Cache.Flag, &Cache.Traffic, &Cache.Usage)

		if Error != nil {
			fmt.Println(">> AccountVerifyVLESS-Error-1:", AccountKey, Error)

			return false
		}

		Cache.Refresh = uint32(time.Now().Unix()) + 5

		// Cache.UserMap = make(map[string]uint32)

		AccountMap[AccountKey] = Cache
	}

	if Cache.Refresh < uint32(time.Now().Unix()) {
		Error := DB.QueryRow("SELECT `day`, `time`, `flag`, `traffic`, `usage` FROM `subscription` WHERE `uuid` = ? LIMIT 1;", AccountKey).Scan(&Cache.Day, &Cache.Time, &Cache.Flag, &Cache.Traffic, &Cache.Usage)

		if Error != nil {
			fmt.Println(">> AccountVerifyVLESS-Error-2:", AccountKey, Error)

			return false
		}

		Cache.Refresh = uint32(time.Now().Unix()) + 5
	}

	if Cache.Flag != "enable" {
		return false
	}

	if Cache.Traffic < Cache.Usage {
		return false
	}

	if Cache.Time == 0 {
		var Time = uint32(time.Now().Unix()) + (Cache.Day * 86400)

		_, Error := DB.Exec("UPDATE `subscription` SET `time` = ? WHERE `uuid` = ? LIMIT 1;", Time, AccountKey)

		if Error != nil {
			fmt.Println(">> AccountVerify-Error-3:", AccountKey, Error)

			return false
		}

		Cache.Time = Time
	} else if Cache.Time < uint32(time.Now().Unix()) {
		return false
	}

	/*
		if Cache.User > 0 {
			IP := AccountIP.String()[:strings.Index(AccountIP.String(), ":")]

			IPCount, OK := Cache.UserMap[IP]

			if !OK {
				if len(Cache.UserMap) > int(Cache.User) {
					// fmt.Println(">> AccountVerify-User:", AccountKey, len(Cache.UserMap), Cache.User)

					return false
				}

				IPCount = 0
			}

			Cache.UserMap[IP] = IPCount + 1
		}
	*/

	return true
}

func AccountUpdateVLESS(AccountUUID []byte, AccountIP net.Addr, CounterUpload int64, CounterDownload int64, Ratio string) {
	AccountMapMutex.Lock()
	defer AccountMapMutex.Unlock()

	AccountKey, AccountError := uuid.ParseBytes(AccountUUID[:])

	if AccountError != nil {
		return
	}

	Cache, OK := AccountMap[AccountKey.String()]

	if OK {
		/*
			if Cache.User > 0 {
				IP := AccountIP.String()[:strings.Index(AccountIP.String(), ":")]

				IPCount, OK := Cache.UserMap[IP]

				if OK {
					if IPCount == 1 {
						delete(Cache.UserMap, IP)
					} else {
						Cache.UserMap[IP] = IPCount - 1
					}
				}
			}
		*/

		Cache.Trace += uint32(CounterUpload + CounterDownload)

		RatioAsFloat, err := strconv.ParseFloat(Ratio, 32)

		if err != nil {
			RatioAsFloat = 1
		}

		RatioMB := 1000000 * RatioAsFloat

		var UsageAsMB = Cache.Trace / uint32(RatioMB)

		if UsageAsMB > 0 {
			_, Error := DB.Exec("UPDATE `subscription` SET `usage` = `usage` + ? WHERE `uuid` = ? LIMIT 1;", UsageAsMB, AccountKey.String())

			if Error != nil {
				fmt.Println(">> AccountUpdateVLESS:", AccountKey.String(), Error)

				return
			}

			Cache.Trace -= (UsageAsMB * uint32(RatioMB))

			Cache.Usage += UsageAsMB
		}
	}
}
