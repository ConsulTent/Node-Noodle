package main

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/rburmorrison/go-argue"
	"github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"
	//	lMail "github.com/zbindenren/logrus_mail"
	"log/syslog"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ConsulTent/gomail"
	ql "modernc.org/ql"
	_ "github.com/mattn/go-sqlite3"
)

type cmdline struct {
	Alerts    string `init:"a" help:"Alert email setup.  Format: \"HOST;PORT;FROM;TO[;USERNAME;PASSWORD]\""`
	Coins     bool   `init:"c" help:"List supported coins and exit."`
	CoinStart string `init:"s" help:"[optional]: Coin start command.  If not specified, alert only."`
	CoinStop  string `init:"t" help:"[optional]: Coin stop command.  If not specified, alert only."`
	TimeDiff  int    `init:"T" help:"[optional]: Block time or time differential.  Default is coin's configured block time (varies)."`
	Failures  int    `init:"f" help:"Number of consecutive failures before we care. [default: 5]"`
	Pid       string `init:"p" help:"[optional]: Specify a pid to write to."`
	Net				bool	 `init:"n" help:"Do not query any external resource.  Rely only on local data. [-n and -i are mutually exclusive]"`
	Inet		  bool	 `init:"i" help:"Ignore local data for alerts, and rely only on Insight/External data. [-n and -i are mutually exclusive]"`
	Offset    int    `init:"o" help:"Time multiplier offset for alert. [default: 3]"`
	Verbose   bool   `init:"v" help:"Verbose mode.  Report Average and Max drift warnings."`
	Daemon    bool   `init:"D" help:"Run with daemon compatibility."`
	Debug     bool   `init:"d" help:"Turn on debug. DO NOT USE unless instructed."`
	Version   bool   `init:"V" help:"Version info."`
}

const pver = "1.1.0"

var gitver = "undefined"

// DEBUG true|false
var DEBUG = false

// timediff max allowed time drift in seconds between blocks set to 5m
var timediff int64
var alert_time int64

var log = logrus.New()

var failures int
var alert_failures int
var alert_time_offset int64

var Coin GenericCoin

var database ql.db

func main() {
	runtime.GOMAXPROCS(2)
	var lastdrift int64
	var avgdrift int64
	var maxdrift int64
	var lastblocktime int64
	var msg string

	var do_restart bool
	var restarted bool

	var cmds cmdline
	agmt := argue.NewEmptyArgumentFromStruct(&cmds)
	agmt.Dispute(true)

	if cmds.Debug == true {
		DEBUG = true
	}

//CoinStart & CoinStop
	if len(cmds.CoinStart) > 0 || len(cmds.CoinStop) > 0 {
		if len(cmds.CoinStart) == 0 {
			log.Warn("You haven't specified a start command.  Assuming external startup is implemented.")
		}

		if len(cmds.CoinStop) == 0 {
			log.Fatal("You must specify both stop AND start commands.")
		}
		do_restart = true
	} else {
		do_restart = false
	}
// -- end CoinStart & CoinStop

// Check for -n and -i conflicts
 if cmds.Inet && cmds.Net {
	 log.Fatal("-n and -i are mutually exclusive.  Please choose only one.")
 }

	//Get (and write?) Pid
	cpid := os.Getpid()
	if len(cmds.Pid) > 0 {
		log.Debug(fmt.Sprintf("PID File: %s, current pid: %d", cmds.Pid, cpid))
		WriteToFile(cmds.Pid, fmt.Sprintf("%d\n", cpid))
	}

	//  fmt.Println(fmt.Sprintf("Alerts: %s, split len: %d, data: %s",cmds.Alerts,len(strings.Split(cmds.Alerts,":")),strings.Split(cmds.Alerts,":")))

	setLogger(cmds.Daemon)

	if cmds.Version == true {
		log.Exit(0)
	}

	if cmds.Failures > 0 {
		alert_failures = cmds.Failures
	} else {
		alert_failures = 5
	}
	failures = 0

	if cmds.Offset > 0 {
		alert_time_offset = int64(cmds.Offset)
	} else {
		alert_time_offset = 3
	}

	if cmds.Coins == true {
		ShowCoins()
		os.Exit(0)
	}

	log.Debug("Autodetecting")
	err := DetectCoin()

	if err == false {
		log.Fatal("No supported coin detected.")
	} else {
		log.Info(fmt.Sprintf("Detected: %s\n", Coin.Name))
	}

	// Starting QL storage

	if debug[0] == true {
		dir, err := ioutil.TempDir("", "ql-debug")
		if err != nil {
			log.Fatal(fmt.Sprintf("%s", err))
		}
		database, err = ql.OpenFile(filepath.Join(dir, fmt.Sprintf("%s.db", Coin.Name)))
		if err != nil {
			log.Fatal(fmt.Sprintf("%s", err))
			panic(err)
		}
	} else {
		database, err = ql.OpenMem()
		if err != nil {
			log.Fatal(fmt.Sprintf("%s", err))
			panic(err)
		}
	}

	// Main loop and background here

	for { // Main Loop

		InitCoin()
		// Detect terrible failures here
		if Coin.Detected == false {
			msg := fmt.Sprintf("Fatal Error reading data. Most likely %s is down!", Coin.Name)
			if len(cmds.Alerts) > 0 {
				go smtpSendMail(cmds.Alerts, msg)
			}
			log.Error(msg)
			if do_restart == true {
				log.Warn("Attempting to recover server.")
				alert_time = int64(time.Now().Unix())
				restarted = RestartCoin(Coin.Tag, cmds.CoinStart, cmds.CoinStop)
				if restarted == false {
					log.Error(fmt.Sprintf("Failed to restart: %s", strings.ToUpper(Coin.Tag)))
				} else {
					log.Warn(fmt.Sprintf("Restarted: %s", strings.ToUpper(Coin.Tag)))
				}
			}
			log.Warn(fmt.Sprintf("Drifting off to sleep for %s, in the hopes server recovers.", time.Duration(cmds.TimeDiff)*time.Second))
			time.Sleep(time.Duration(cmds.TimeDiff) * time.Second)
			break
		} else {
			if cmds.TimeDiff > 0 {
				timediff = int64(cmds.TimeDiff)
			} else {
				timediff = Coin.BlockTime
			}
			log.Debug(fmt.Sprintf("timediff: %d\n", timediff))
		}

		log.Debug("Entering Sqlite3")
		SaveToQL(Coin.Tag, cmds.Debug)
		lastdrift, avgdrift, maxdrift, lastblocktime = GetBlockDrifts(Coin.Tag, cmds.Debug)

		if cmds.Verbose == true {
			if time.Duration(maxdrift) > time.Duration(timediff) {
				log.Info(fmt.Sprintf("%s Maxdrift is %s, and over %s", strings.ToUpper(Coin.Tag), time.Duration(maxdrift)*time.Second, time.Duration(timediff)*time.Second))
			} else {
				log.Info(fmt.Sprintf("%s Maxdrift is %s", strings.ToUpper(Coin.Tag), time.Duration(maxdrift)*time.Second))
			}

			if time.Duration(avgdrift) > time.Duration(timediff) {
				log.Info(fmt.Sprintf("%s Avgdrift is %s, and over %s", strings.ToUpper(Coin.Tag), time.Duration(avgdrift)*time.Second, time.Duration(timediff)*time.Second))
			} else {
				log.Info(fmt.Sprintf("%s Avgdrift is %s", strings.ToUpper(Coin.Tag), time.Duration(avgdrift)*time.Second))
			}

		}

// *** Insight ***
	if cmds.Net == false {
	  go PrintBlocksBehind(cmds.Verbose)
	}
// *** Insight ***

		if lastdrift > timediff {
			log.Warn(fmt.Sprintf("%s Lastdrift was over %s by %s", strings.ToUpper(Coin.Tag), time.Duration(timediff)*time.Second, time.Duration(lastdrift)*time.Second-time.Duration(timediff)*time.Second))
		}

		if (int64(time.Now().Unix() - lastblocktime)) > (timediff * alert_time_offset) {
			failures++
			log.Debug(fmt.Sprintf("Failures: %d. We alert on %d concurrent failures.", failures, alert_failures))
			if failures >= alert_failures {
			 if cmds.Net == false {
				msg = fmt.Sprintf("%s Last Block time is %s (%s behind %s), that's over %d x our %s timediff!\nWe are %d blocks behind the network.", strings.ToUpper(Coin.Tag),
					time.Unix(lastblocktime, 0).Format("Mon Jan _2 15:04:05 2006"), time.Duration(time.Now().Unix()-lastblocktime)*time.Second, time.Now().Format("Mon Jan _2 15:04:05 2006"),
					 alert_time_offset, time.Duration(timediff)*time.Second,Coin.InsightBlocks - Coin.Blocks)
			 } else {
				 msg = fmt.Sprintf("%s Last Block time is %s (%s behind %s), that's over %d x our %s timediff!", strings.ToUpper(Coin.Tag),
 					time.Unix(lastblocktime, 0).Format("Mon Jan _2 15:04:05 2006"), time.Duration(time.Now().Unix()-lastblocktime)*time.Second, time.Now().Format("Mon Jan _2 15:04:05 2006"),
 					 alert_time_offset, time.Duration(timediff)*time.Second)
			 }
				log.Error(msg)
				// Fire off an alert! ONLY if alerts are set!
				if len(cmds.Alerts) > 0 {
					if cmds.Inet == true && (Coin.InsightBlocks - Coin.Blocks != 0) {
						msg = fmt.Sprintf("%s is %d blocks behind the network.", strings.ToUpper(Coin.Tag), Coin.InsightBlocks - Coin.Blocks)
						go smtpSendMail(cmds.Alerts, msg)
					} else {
					  go smtpSendMail(cmds.Alerts, msg)
				  }
				}

				if do_restart == true {
					alert_time = int64(time.Now().Unix())
					restarted = RestartCoin(Coin.Tag, cmds.CoinStart, cmds.CoinStop)
					if restarted == false {
						log.Error(fmt.Sprintf("Failed to restart: %s", strings.ToUpper(Coin.Tag)))
					} else {
						log.Warn(fmt.Sprintf("Restarted: %s", strings.ToUpper(Coin.Tag)))
					}
				}
			} else {
				log.Warn(fmt.Sprintf("%s Last Block time is %s (%s behind %s), that's over %d x our %s timediff!", strings.ToUpper(Coin.Tag),
					time.Unix(lastblocktime, 0).Format("Mon Jan _2 15:04:05 2006"), time.Duration(time.Now().Unix()-lastblocktime)*time.Second, time.Now().Format("Mon Jan _2 15:04:05 2006"), alert_time_offset, time.Duration(timediff)*time.Second))
			}
		} else {
			failures = 0
		}

		//Sleeps
		log.Debug(fmt.Sprintf("Drifting off to sleep for %s", time.Duration(timediff)*time.Second))
		time.Sleep(time.Duration(timediff) * time.Second)

		if SqliteHousekeeping(Coin.Tag, cmds.Debug) == true && cmds.Verbose == true {
			log.Info("Housekeeping cleared out data.")
		}

	} // Loop

}

func RestartCoin(coin string, start string, stop string) bool {
	var output string
	var command string

	log.Debug(fmt.Sprintf("Stopping %s with %s", strings.ToUpper(coin), stop))

	command = fmt.Sprintf("%s", stop)

	output = exe_cmd(command)

	log.Info(output)

	//log.Debug("Sleep for %s",time.Duration(10) * time.Second)
	log.Info(fmt.Sprintf("Sleeping for %s", time.Duration(300)*time.Second))
	time.Sleep(time.Duration(300) * time.Second)

	if len(stop) > 0 {
		log.Debug(fmt.Sprintf("Starting %s with %s", strings.ToUpper(coin), start))

		command = fmt.Sprintf("%s", start)

		output = exe_cmd(command)

		log.Info(output)
	}

	return true
}

// Converting to QL
func SaveToQL(coin string, debug ...bool) bool {
	
	log.Debug("Entering QL:SaveToQL")
	
	rs, _, err := database.Run(NewRWCtx(), `
	CREATE TABLE IF NOT EXISTS blocks (id INTEGER PRIMARY KEY, Coin TEXT, Blocks BIGINT, BlockTime BIGINT, CaptureTime BIGINT);
	SELECT id, Coin, Blocks, BlockTime, CaptureTime FROM blocks ORDER BY CaptureTime DESC LIMIT 1;
	`, 1)
	if err != nil {
		log.Fatal(fmt.Sprintf("%s", err))
		panic(err)
	}

	log.Debug("QL Read Data")
	var id int
	var blocks int64
	var blocktime int64
	var capturetime int64

	for rows.Next() {
		log.Debug("QL read row")
		rows.Scan(&id, &coin, &blocks, &blocktime, &capturetime)
		//		fmt.Println(strconv.Itoa(id) + ": " + coin + " " + strconv.FormatInt(blocks,10) + " " + strconv.FormatInt(blocktime,10) + " " + strconv.FormatInt(capturetime,10))
	}
	if blocktime == Coin.Time {
		log.Debug("Blocktimes are equal, skipping insert.")
	} else {
		statement, _ = database.Prepare("INSERT INTO blocks (Coin, Blocks, BlockTime, CaptureTime) VALUES (?, ?, ?, ?)")
		log.Debug("Insert prepared")
		log.Debug("executing Sqlite3 statement")
		log.Debug(fmt.Sprintf("INSERT DATA: %s %d %d %d", Coin.Tag, Coin.Blocks, Coin.Time, Coin.CaptureTime))
		statement.Exec(Coin.Tag, Coin.Blocks, Coin.Time, Coin.CaptureTime)
		log.Debug("Sqlite3 Inserted Data")
	}
	//database.Close()
	return true
}

// GetBlockDrifts calculate and return last, average, and max drift, as well as last block time.
// Calculate block drift from time.Now().Unix() NOT from previous block
func GetBlockDrifts(coin string, debug ...bool) (last int64, avg int64, max int64, block int64) {
	var database *sql.DB
	var maxdrift int64
	var drift int64
	var lastdrift int64
	var blocktime int64
	var blocktimes [10]int64
	var blockdrifts [10]int64
	var totaltimes int64
	var i int

	log.Debug("Entering Sqlite3:GetMaxBlockDrift")

	if debug[0] == true {
		database, _ = sql.Open("sqlite3", fmt.Sprintf("file:%s.db?cache=shared", coin))
	} else {
		database, _ = sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", coin))
	}
	database.SetMaxOpenConns(1)
	database.SetConnMaxLifetime(0)
	database.SetMaxIdleConns(4)
	rows, _ := database.Query("SELECT q.BlockTime FROM (SELECT BlockTime,CaptureTime FROM blocks WHERE Coin like \"" + coin + "\" ORDER BY CaptureTime DESC LIMIT 10) q ORDER BY q.CaptureTime ASC")
	log.Debug("Sqlite3 Read Data")

	i = 0
	totaltimes = 0
	drift = 0
	maxdrift = 0

	// Last row is the last value also block time
	for rows.Next() {

		log.Debug("Sqlite3 read row")

		rows.Scan(&blocktime)
		blocktimes[i] = blocktime
		totaltimes = totaltimes + blocktime
		drift = int64(time.Now().Unix() - blocktime)
		blockdrifts[i] = drift

		if drift > maxdrift {
			maxdrift = drift
		}

		log.Debug(fmt.Sprintf("Blocktime: %s", strconv.FormatInt(blocktime, 10)))
		i++
	}
	lastdrift = drift
	drift = int64(time.Now().Unix() - (totaltimes / int64(i)))
	log.Debug(fmt.Sprintf("i: %d, lastdrift: %s, avgdrift: %s, maxdrift: %s", i, time.Duration(lastdrift)*time.Second, time.Duration(drift)*time.Second, time.Duration(maxdrift)*time.Second))

	//database.Close()
	return lastdrift, drift, maxdrift, blocktime
}

func SqliteHousekeeping(coin string, debug ...bool) bool {
	var database *sql.DB
	var maxrows = 10
	log.Debug("Entering Sqlite3:SqliteHousekeeping")

	if debug[0] == true {
		database, _ = sql.Open("sqlite3", fmt.Sprintf("file:%s.db?cache=shared", coin))
	} else {
		database, _ = sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", coin))
	}
	database.SetMaxOpenConns(1)
	database.SetConnMaxLifetime(0)
	database.SetMaxIdleConns(4)
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS blocks (id INTEGER PRIMARY KEY, Coin TEXT, Blocks BIGINT, BlockTime BIGINT, CaptureTime BIGINT)")
	statement.Exec()

	rows, _ := database.Query("SELECT Count(id) FROM blocks")
	log.Debug("Sqlite3 Read Data")
	var count int

	for rows.Next() {
		log.Debug("Sqlite3 read row")
		rows.Scan(&count)
	}

	if count > (maxrows * 2) {
		log.Debug(fmt.Sprintf("Will be deleting max %d rows", maxrows))
		rows, _ = database.Query(fmt.Sprintf("DELETE from blocks order by id ASC limit %d", maxrows))
		log.Debug(fmt.Sprintf("DELETE from blocks order by id ASC limit %d", maxrows))
		log.Debug("Sqlite3 Deleted Data")
		//		database.Close()
		return true
	}
	//	database.Close()
	return false
}

func IsInTimeRange(svalue int64, evalue int64, timeout int) bool {
	delta := int(svalue - evalue)
	return -timeout < delta && delta < timeout
}

func InTimeRange(tvalue int64, timeout int) bool {
	delta := int(time.Now().Unix() - tvalue)
	return -timeout < delta && delta < timeout
}

// setLogger configure both logging and alerting
func setLogger(c bool) {

	if c == true {
		hook, err := lSyslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
		log.SetReportCaller(false)
		if err == nil {
			log.Hooks.Add(hook)
			log.SetFormatter(&logrus.JSONFormatter{})
			log.Info(fmt.Sprintf("Node Noodle (c) 2019 ConsulTent Ltd. (http://consultent.ltd) v%s-%s", pver, gitver))
			log.Info("Donations accepted [Firo]: aGoK6MF87K2SgT7cnJFhSWt7u2cAS5m18p\n\n")
		}
	} else {
		log.SetFormatter(&logrus.TextFormatter{})
		log.SetOutput(os.Stdout)
		fmt.Printf("Node Noodle (c) 2019 ConsulTent Ltd. v%s-%s\nhttp://consultent.ltd\n", pver, gitver)
		fmt.Printf("Donations accepted [Firo]: aGoK6MF87K2SgT7cnJFhSWt7u2cAS5m18p\n\n")
	}

	if DEBUG == true {
		log.SetLevel(logrus.DebugLevel)
	}

}

func exe_cmd(cmd string) (outs string) {
	log.Debug(fmt.Sprintf("command is %s", cmd))

	// splitting head => g++ parts => rest of the command
	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:len(parts)]

	out, err := exec.Command(head, parts...).Output()
	if err != nil {
		log.Fatal(fmt.Sprintf("%s", err))
	}
	//  fmt.Printf("%s", out)
	// Need to signal to waitgroup that this goroutine is done

	return fmt.Sprintf("%s", out)
}

func smtpSendMail(e string, msg string) bool {
	var emailConfig = make([]string, 0)
	var Port int
	var Hostname string

	Hostname, _ = os.Hostname()

	//	emailConfig = make([]string,6)

	log.Debug(fmt.Sprintf("Email config: %s", e))
	emailConfig = strings.SplitN(e, ";", 6)

	log.Debug(fmt.Sprintf("emailConfig len: %d", len(emailConfig)))

	if len(emailConfig) == 4 {
		log.Debug(fmt.Sprintf("emailConfig, appending 2 blanks"))
		//		 emailConfig = emailConfig[0:len(emailConfig)+2]
		emailConfig = append(emailConfig, "", "")
		log.Debug(fmt.Sprintf("emailConfig: %s", emailConfig))
	}

	Port, _ = strconv.Atoi(emailConfig[1])

	log.Debug(fmt.Sprintf("Port: %d", Port))

	sender := &gomail.Sender{
		User:   emailConfig[4],
		Passwd: emailConfig[5],
		Host:   emailConfig[0],
		Port:   Port,
	}

	log.Debug("sender.Configure()")
	sender.Configure()

	sw := sender.NewSendWorker(
		emailConfig[2],
		emailConfig[3],
		"Node Noodle Alert",
	)

	fromName := strings.Split(emailConfig[3], "@")

	err := sw.ParseTemplate(
		`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
	        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
	<html>

	</head>

	<body>
	<p>
	    Hello {{.Name}}, from host {{.HOST}},
	    <B>{{.ALERT}}</B>
			<p>This message brought to you by <a href="http://consultent.ltd/">Node Noodle</a>.
	</p>

	</body>

	</html>`,
		map[string]string{
			"Name":  fromName[0],
			"ALERT": msg,
			"HOST":  Hostname,
		},
	)

	if err != nil {
		log.Warn(err)
	}

	err = sw.SendEmail()

	if err != nil {
		log.Warn(err)
	}
	log.Info("Email Alert Sent!")
	return true
}

// WriteToFile will print any string of text to a file safely by
// checking for errors and syncing at the end.
func WriteToFile(filename string, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		log.Fatal(err)
	}
	return file.Sync()
}
