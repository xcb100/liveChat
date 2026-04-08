package cmd

import (
	"goflylivechat/common"
	"goflylivechat/models"
	"goflylivechat/tools"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Initialize database and import data", // More precise description
	Run: func(cmd *cobra.Command, args []string) {
		install()
	},
}

func install() {
	if models.IsSetupReady() {
		log.Println("Database schema is already initialized")
		os.Exit(1)
	}

	sqlFile := "import.sql"
	dataExists, _ := tools.IsFileExist(sqlFile)
	if !dataExists {
		log.Println("Database import file import.sql not found")
		os.Exit(1)
	}

	mysqlConfig := common.GetMysqlConf()
	importer := tools.ImportSqlTool{
		SqlPath:  sqlFile,
		Username: mysqlConfig.Username,
		Password: mysqlConfig.Password,
		Server:   mysqlConfig.Server,
		Port:     mysqlConfig.Port,
		Database: mysqlConfig.Database,
	}
	if err := importer.ImportSql(); err != nil {
		log.Printf("Database initialization failed: %v\n", err)
		os.Exit(1)
	}

	installFile, err := os.OpenFile("./install.lock", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Printf("Failed to create lock file: %v\n", err)
		os.Exit(1)
	}
	defer installFile.Close()

	_, err = installFile.WriteString("gofly live chat installation complete")
	if err != nil {
		log.Printf("Failed to write lock file: %v\n", err)
		os.Exit(1)
	}

	log.Println("Database initialization completed successfully")
}
