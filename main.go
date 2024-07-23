package main

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"syscall"
	"time"

	"github.com/araddon/dateparse"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/xiangxn/listener/config"
	"github.com/xiangxn/listener/monitor"
	"github.com/xiangxn/listener/stats"
	"github.com/xiangxn/listener/strategies"
	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
)

var conf config.Configuration

func main() {
	var rootCmd = &cobra.Command{
		Use: "listener",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			configFile, _ := cmd.Flags().GetString("config")
			conf = config.GetConfig(configFile)
		},
	}
	rootCmd.PersistentFlags().StringP("config", "c", "config.yaml", "Configuration file name")

	var arbCmd = &cobra.Command{
		Use:   "arb",
		Short: "Arbitrage command",
		Run: func(cmd *cobra.Command, args []string) {
			arbitrage(conf)
		},
	}

	var statsCmd = &cobra.Command{
		Use:   "stats",
		Short: "statistics command",
		Run: func(cmd *cobra.Command, args []string) {
			day, _ := cmd.Flags().GetInt("day")
			start, _ := cmd.Flags().GetString("start")
			end, _ := cmd.Flags().GetString("end")
			simulate, _ := cmd.Flags().GetBool("simulate")
			statsFun(conf, day, start, end, simulate)
		},
	}
	statsCmd.Flags().IntP("day", "D", 1, "Statistics of the last few days")
	statsCmd.Flags().StringP("start", "S", "", "The statistics start date")
	statsCmd.Flags().StringP("end", "E", "", "End date of statistics")
	statsCmd.Flags().BoolP("simulate", "M", false, "When true, only simulated transactions are counted; when false, only real transactions are counted")

	var decryptCmd = &cobra.Command{
		Use:   "crypto",
		Short: "Crypto command",
		Run: func(cmd *cobra.Command, args []string) {
			decrypt, _ := cmd.Flags().GetBool("decrypt")
			encrypt, _ := cmd.Flags().GetBool("encrypt")
			if decrypt {
				crypto(false, args)
			} else if encrypt {
				crypto(true, args)
			}
		},
	}
	decryptCmd.Flags().BoolP("decrypt", "D", false, "Decrypt the characters specified by the parameter")
	decryptCmd.Flags().BoolP("encrypt", "E", false, "Encrypt the characters specified by the parameter")

	rootCmd.AddCommand(arbCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(decryptCmd)
	rootCmd.Execute()
}

func crypto(e bool, args []string) {
	fmt.Print("Enter password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Println("Error reading password:", err)
		return
	}
	password := string(passwordBytes)
	key := sha256.Sum256([]byte(password))
	if e {
		ciphertext, err := tools.Encrypt([]byte(args[0]), key[:])
		if err != nil {
			fmt.Println("Error encrypting:", err)
			return
		}
		b32 := base32.StdEncoding.EncodeToString(ciphertext)
		fmt.Printf("Ciphertext: %s\n", b32)
	} else {
		ciphertext, err := base32.StdEncoding.DecodeString(args[0])
		if err != nil {
			fmt.Println("Base32 error decrypting:", err)
			return
		}
		decryptedText, err := tools.Decrypt(ciphertext, key[:])
		if err != nil {
			fmt.Println("Error decrypting:", err)
			return
		}
		fmt.Printf("Decrypted text: %s\n", decryptedText)
	}
}

func statsFun(conf config.Configuration, day int, start, end string, simulate bool) {
	sts := stats.New(conf)
	if start != "" && end != "" {
		sData, err := dateparse.ParseAny(start)
		if err != nil {
			panic(err)
		}
		eData, err := dateparse.ParseAny(end)
		if err != nil {
			panic(err)
		}
		sts.SearchTransacttion(simulate, sData, eData)
	} else {
		eData := time.Now()
		sData := eData.AddDate(0, 0, -day)
		sts.SearchTransacttion(simulate, sData, eData)
	}
}

func arbitrage(conf config.Configuration) {
	fmt.Print("Enter password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Println("Error reading password:", err)
		return
	}
	password := string(passwordBytes)

	err = godotenv.Load()
	if err != nil {
		panic(err)
	}

	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	if conf.Debug {
		l.Level = logrus.DebugLevel
	}

	opt := &dt.Options{
		Cfg:     conf,
		Handler: &strategies.MovingBrick{},
		Logger:  l,
		Cipher:  sha256.Sum256([]byte(password)),
	}

	monitor, err := monitor.New(opt)
	if err != nil {
		panic(err)
	}
	monitor.Run()
	monitor.Cancel()
}
