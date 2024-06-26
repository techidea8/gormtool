package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DBType database type
type DBType string

const (
	// dbMySQL Gorm Drivers mysql || postgres || sqlite || sqlserver
	dbMySQL      DBType = "mysql"
	dbPostgres   DBType = "postgres"
	dbSQLite     DBType = "sqlite"
	dbSQLServer  DBType = "sqlserver"
	dbClickHouse DBType = "clickhouse"
)
const (
	defaultQueryPath = "./dao/query"
)

type JSONNSTYPE string

const (
	JSONNSCAMEL  JSONNSTYPE = "camel"
	JSONNSPASCAL JSONNSTYPE = "pascal"
	JSONNSUNDER  JSONNSTYPE = "under"
)

const VERSION = "gormtool@20240614"

// CmdParams is command line parameters
type CmdParams struct {
	DSN                 string   `yaml:"dsn"`                 // consult[https://gorm.io/docs/connecting_to_the_database.html]"
	DB                  string   `yaml:"db"`                  // input mysql or postgres or sqlite or sqlserver. consult[https://gorm.io/docs/connecting_to_the_database.html]
	Tables              []string `yaml:"tables"`              // enter the required data table or leave it blank
	OnlyModel           bool     `yaml:"onlyModel"`           // only generate model
	OutPath             string   `yaml:"outPath"`             // specify a directory for output
	OutFile             string   `yaml:"outFile"`             // query code file name, default: gen.go
	WithUnitTest        bool     `yaml:"withUnitTest"`        // generate unit test for query code
	ModelPkgName        string   `yaml:"modelPkgName"`        // generated model code's package name
	FieldNullable       bool     `yaml:"fieldNullable"`       // generate with pointer when field is nullable
	FieldCoverable      bool     `yaml:"fieldCoverable"`      // generate with pointer when field has default value
	FieldWithIndexTag   bool     `yaml:"fieldWithIndexTag"`   // generate field with gorm index tag
	FieldWithTypeTag    bool     `yaml:"fieldWithTypeTag"`    // generate field with gorm column type tag
	FieldSignable       bool     `yaml:"fieldSignable"`       // detect integer field's unsigned type, adjust generated data type
	JsonTagNameStrategy string   `yaml:"jsonTagNameStrategy"` // detect integer field's unsigned type, adjust generated data type
	TablePrefix         string   `yaml:"tablePrefix"`
	SingularTable       bool     `yaml:"singularTable"`
}

func (c *CmdParams) revise() *CmdParams {
	if c == nil {
		return c
	}
	if c.DB == "" {
		c.DB = string(dbMySQL)
	}
	if c.OutPath == "" {
		c.OutPath = defaultQueryPath
	}
	if len(c.Tables) == 0 {
		return c
	}

	tableList := make([]string, 0, len(c.Tables))
	for _, tableName := range c.Tables {
		_tableName := strings.TrimSpace(tableName) // trim leading and trailing space in tableName
		if _tableName == "" {                      // skip empty tableName
			continue
		}
		tableList = append(tableList, _tableName)
	}
	c.Tables = tableList
	return c
}

// YamlConfig is yaml config struct
type YamlConfig struct {
	Version  string     `yaml:"version"`  //
	Database *CmdParams `yaml:"database"` //
}

// connectDB choose db type for connection to database
func connectDB(t DBType, dsn string, s schema.NamingStrategy) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("dsn cannot be empty")
	}

	switch t {
	case dbMySQL:
		return gorm.Open(mysql.Open(dsn), &gorm.Config{
			NamingStrategy: s,
		})
	case dbPostgres:
		return gorm.Open(postgres.Open(dsn), &gorm.Config{
			NamingStrategy: s,
		})
	case dbSQLite:
		return gorm.Open(sqlite.Open(dsn), &gorm.Config{
			NamingStrategy: s,
		})
	case dbSQLServer:
		return gorm.Open(sqlserver.Open(dsn), &gorm.Config{
			NamingStrategy: s,
		})
	case dbClickHouse:
		return gorm.Open(clickhouse.Open(dsn), &gorm.Config{
			NamingStrategy: s,
		})
	default:
		return nil, fmt.Errorf("unknow db %q (support mysql || postgres || sqlite || sqlserver for now)", t)
	}
}

// genModels is gorm/gen generated models
func genModels(g *gen.Generator, db *gorm.DB, tables []string) (models []interface{}, err error) {
	if len(tables) == 0 {
		// Execute tasks for all tables in the database
		tables, err = db.Migrator().GetTables()
		if err != nil {
			return nil, fmt.Errorf("GORM migrator get all tables fail: %w", err)
		}
	}

	// Execute some data table tasks
	models = make([]interface{}, len(tables))
	for i, tableName := range tables {
		models[i] = g.GenerateModel(tableName)
	}
	return models, nil
}

// parseCmdFromYaml parse cmd param from yaml
func parseCmdFromYaml(path string) *CmdParams {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("parseCmdFromYaml fail %s", err.Error())
		return nil
	}
	defer file.Close() // nolint
	var yamlConfig YamlConfig
	if err = yaml.NewDecoder(file).Decode(&yamlConfig); err != nil {
		log.Fatalf("parseCmdFromYaml fail %s", err.Error())
		return nil
	}
	return yamlConfig.Database
}

var version = flag.Bool("v", false, "version of gormtool")

// argParse is parser for cmd
func argParse() *CmdParams {
	// choose is file or flag
	genPath := flag.String("c", "", "is path for gen.yml")

	dsn := flag.String("dsn", "", "consult[https://gorm.io/docs/connecting_to_the_database.html]")
	db := flag.String("db", string(dbMySQL), "input mysql|postgres|sqlite|sqlserver|clickhouse. consult[https://gorm.io/docs/connecting_to_the_database.html]")
	tableList := flag.String("tables", "", "enter the required data table or leave it blank")
	onlyModel := flag.Bool("onlyModel", false, "only generate models (without query file)")
	outPath := flag.String("outPath", defaultQueryPath, "specify a directory for output")
	outFile := flag.String("outFile", "", "query code file name, default: gen.go")
	withUnitTest := flag.Bool("withUnitTest", false, "generate unit test for query code")
	modelPkgName := flag.String("modelPkgName", "", "generated model code's package name")
	fieldNullable := flag.Bool("fieldNullable", false, "generate with pointer when field is nullable")
	fieldCoverable := flag.Bool("fieldCoverable", false, "generate with pointer when field has default value")
	fieldWithIndexTag := flag.Bool("fieldWithIndexTag", false, "generate field with gorm index tag")
	fieldWithTypeTag := flag.Bool("fieldWithTypeTag", false, "generate field with gorm column type tag")
	fieldSignable := flag.Bool("fieldSignable", false, "detect integer field's unsigned type, adjust generated data type")
	jsonTagNameStrategy := flag.String("jsonTagNameStrategy", "under", "camel/pascal/under,use test_field camel:testField,pascal:TestField,under:test_field")
	singularTable := flag.Bool("singularTable", false, "singularTable or not ,use bool")
	tablePrefix := flag.String("tablePrefix", "", "prefix of table")
	flag.Parse()

	if *genPath != "" { //use yml config
		return parseCmdFromYaml(*genPath)
	}
	var cmdParse CmdParams
	// cmd first
	if *dsn != "" {
		cmdParse.DSN = *dsn
	}
	if *db != "" {
		cmdParse.DB = *db
	}
	if *tableList != "" {
		cmdParse.Tables = strings.Split(*tableList, ",")
	}
	if *onlyModel {
		cmdParse.OnlyModel = true
	}
	if *outPath != "" {
		cmdParse.OutPath = *outPath
	}
	if *outFile != "" {
		cmdParse.OutFile = *outFile
	}
	if *withUnitTest {
		cmdParse.WithUnitTest = *withUnitTest
	}
	if *modelPkgName != "" {
		cmdParse.ModelPkgName = *modelPkgName
	}
	if *fieldNullable {
		cmdParse.FieldNullable = *fieldNullable
	}
	if *fieldCoverable {
		cmdParse.FieldCoverable = *fieldCoverable
	}
	if *fieldWithIndexTag {
		cmdParse.FieldWithIndexTag = *fieldWithIndexTag
	}
	if *fieldWithTypeTag {
		cmdParse.FieldWithTypeTag = *fieldWithTypeTag
	}
	if *fieldSignable {
		cmdParse.FieldSignable = *fieldSignable
	}
	cmdParse.JsonTagNameStrategy = *jsonTagNameStrategy

	cmdParse.SingularTable = *singularTable

	if *tablePrefix != "" {
		cmdParse.TablePrefix = *tablePrefix
	}
	return &cmdParse
}
func FirstUpper(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// FirstLower 字符串首字母小写
func FirstLower(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}
func main() {
	// cmdParse
	config := argParse().revise()
	if config == nil {
		log.Fatalln("parse config fail")
	}

	if *version {
		log.Println(VERSION)
		return
	}
	db, err := connectDB(DBType(config.DB), config.DSN, schema.NamingStrategy{
		TablePrefix:   config.TablePrefix,
		SingularTable: config.SingularTable,
	})
	if err != nil {
		log.Fatalln("connect db server fail:", err)
	}

	g := gen.NewGenerator(gen.Config{
		OutPath:           config.OutPath,
		OutFile:           config.OutFile,
		ModelPkgPath:      config.ModelPkgName,
		WithUnitTest:      config.WithUnitTest,
		FieldNullable:     config.FieldNullable,
		FieldCoverable:    config.FieldCoverable,
		FieldWithIndexTag: config.FieldWithIndexTag,
		FieldWithTypeTag:  config.FieldWithTypeTag,
		FieldSignable:     config.FieldSignable,
	})

	// g.WithTableNameStrategy(func(tableName string) (targetTableName string) {
	// 	if config.SingularTable {
	// 		return tableName
	// 	} else {
	// 		return tableName
	// 	}
	// })
	g.WithJSONTagNameStrategy(func(columnName string) (tagContent string) {
		if config.JsonTagNameStrategy == string(JSONNSCAMEL) {
			// test_z3
			arr := strings.Split(columnName, "_")
			for i, v := range arr {
				if i == 0 {
					arr[i] = FirstLower(v)
				} else {
					arr[i] = FirstUpper(v)
				}
			}

			return strings.Join(arr, "")
		} else if config.JsonTagNameStrategy == string(JSONNSPASCAL) {
			arr := strings.Split(columnName, "_")
			for i, v := range arr {
				arr[i] = FirstUpper(v)
			}
			return strings.Join(arr, "")
		} else {
			return columnName
		}

	})
	g.UseDB(db)

	models, err := genModels(g, db, config.Tables)
	if err != nil {
		log.Fatalln("get tables info fail:", err)
	}

	if !config.OnlyModel {
		g.ApplyBasic(models...)
	}

	g.Execute()
}
