// Copyright 2020-2022 Demian Harvill
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command line launches gRPC MServiceLedger server.
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/gaterace/mledger/pkg/glauth"
	"github.com/gaterace/mledger/pkg/glservice"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	cli := &cli{}

	cmd := &cobra.Command{
		Use:     "invserver",
		PreRunE: cli.setupConfig,
		RunE:    cli.run,
	}

	if err := setupFlags(cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)

	}

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

type cli struct {
	cfg cfg
}

type cfg struct {
	ProjConf    string
	LogFile     string
	CertFile    string
	KeyFile     string
	Tls         bool
	Port        int
	RestPort    int
	DbUser      string
	DbPwd       string
	DbTransport string
	JwtPubFile  string
	CorsOrigin  string
}

func setupFlags(cmd *cobra.Command) error {

	cmd.Flags().String("conf", "conf.yaml", "Path to inventory config file.")
	cmd.Flags().String("log_file", "", "Path to log file.")
	cmd.Flags().String("cert_file", "", "Path to certificate file.")
	cmd.Flags().String("key_file", "", "Path to certificate key file.")
	cmd.Flags().Bool("tls", false, "Use tls for connection.")
	cmd.Flags().Int("port", 50056, "Port for RPC connections")

	cmd.Flags().String("db_user", "", "Database user name.")
	cmd.Flags().String("db_pwd", "", "Database user password.")
	cmd.Flags().String("db_transport", "", "Database transport string.")
	cmd.Flags().String("jwt_pub_file", "", "Path to JWT public certificate.")

	return viper.BindPFlags(cmd.Flags())
}

func (c *cli) setupConfig(cmd *cobra.Command, args []string) error {
	var err error

	viper.SetEnvPrefix("gl")

	viper.AutomaticEnv()

	configFile := viper.GetString("conf")

	viper.SetConfigFile(configFile)

	if err = viper.ReadInConfig(); err != nil {
		// it's ok if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	c.cfg.LogFile = viper.GetString("log_file")
	c.cfg.CertFile = viper.GetString("cert_file")
	c.cfg.KeyFile = viper.GetString("key_file")
	c.cfg.Tls = viper.GetBool("tls")
	c.cfg.Port = viper.GetInt("port")

	c.cfg.DbUser = viper.GetString("db_user")
	c.cfg.DbPwd = viper.GetString("db_pwd")
	c.cfg.DbTransport = viper.GetString("db_transport")
	c.cfg.JwtPubFile = viper.GetString("jwt_pub_file")

	return nil
}

func (c *cli) run(cmd *cobra.Command, args []string) error {
	var err error

	log_file := c.cfg.LogFile
	cert_file := c.cfg.CertFile
	key_file := c.cfg.KeyFile
	tls := c.cfg.Tls
	port := c.cfg.Port
	db_user := c.cfg.DbUser
	db_pwd := c.cfg.DbPwd
	db_transport := c.cfg.DbTransport
	jwt_pub_file := c.cfg.JwtPubFile

	var logWriter io.Writer

	if log_file == "" {
		logWriter = os.Stderr
	} else {
		logfile, _ := os.OpenFile(log_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer logfile.Close()
		logWriter = logfile
	}
	logger := log.NewLogfmtLogger(log.NewSyncWriter(logWriter))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	level.Info(logger).Log("log_file", log_file)
	level.Info(logger).Log("cert_file", cert_file)
	level.Info(logger).Log("key_file", key_file)
	level.Info(logger).Log("tls", tls)
	level.Info(logger).Log("port", port)
	level.Info(logger).Log("db_user", db_user)
	level.Info(logger).Log("db_transport", db_transport)
	level.Info(logger).Log("jwt_pub_file", jwt_pub_file)

	listen_port := ":" + strconv.Itoa(int(port))
	// fmt.Println(listen_port)

	lis, err := net.Listen("tcp", listen_port)
	if err != nil {
		level.Error(logger).Log("what", "net.listen", "error", err)
		os.Exit(1)
	}

	var opts []grpc.ServerOption
	if tls {
		creds, err := credentials.NewServerTLSFromFile(cert_file, key_file)
		if err != nil {
			level.Error(logger).Log("what", "Failed to generate credentials", "error", err)
			os.Exit(1)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}

	s := grpc.NewServer(opts...)

	glService := glservice.NewGlService()

	sqlDb, err := SetupDatabaseConnections(db_user, db_pwd, db_transport)
	if err != nil {
		level.Error(logger).Log("what", "SetupDatabaseConnections", "error", err)
		os.Exit(1)
	}

	glService.SetLogger(logger)
	glService.SetDatabaseConnection(sqlDb)

	// wire up the authorization middleware

	glAuth := glauth.NewLedgerAuth(glService)

	glAuth.SetLogger(logger)

	glAuth.SetPublicKey(jwt_pub_file)
	glAuth.SetDatabaseConnection(sqlDb)
	err = glAuth.NewApiServer(s)
	if err != nil {
		level.Error(logger).Log("what", "NewApiServer", "error", err)
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "starting grpc server")

	err = s.Serve(lis)
	if err != nil {
		level.Error(logger).Log("what", "Serve", "error", err)
	}

	level.Info(logger).Log("msg", "shutting down grpc server")

	return err

}

// Helper to set up the database connection.
func SetupDatabaseConnections(db_user string, db_pwd string, db_transport string) (*sql.DB, error) {
	var sqlDb *sql.DB
	endpoint := db_user + ":" + db_pwd + "@" + db_transport + "/mledger?parseTime=true"
	var err error
	sqlDb, err = sql.Open("mysql", endpoint)
	if err == nil {
		err = sqlDb.Ping()
		if err != nil {
			sqlDb = nil
		}

	}

	return sqlDb, err
}
