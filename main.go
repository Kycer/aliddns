package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/ahmetb/go-linq/v3"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/jasonlvhit/gocron"
)

var CONFIG_PATH = []string{"~/.config/aliddns", "./configs"}

var CONFIG_MAP = make(map[string]*DDNSConfig)

//  主配置文件
type DDNSConfig struct {
	AliAccess *AliAccess
	Domains   *[]Domain
}

type AliAccess struct {
	AccessId  string
	AccessKey string
	Region    string
	Domain    string
}

type Domain struct {
	Rr         string
	DomainType string
	UpdateType string
	Value      string
}

func getIP() string {
	responseClient, errClient := http.Get("https://ipv4.ipw.cn/api/ip/myip")

	if errClient != nil {
		log.Printf("获取外网 IP 失败，请检查网络\n")
		panic(errClient)
	}
	defer responseClient.Body.Close()

	// 获取 http response 的 body
	body, _ := ioutil.ReadAll(responseClient.Body)
	clientIP := string(body)
	return clientIP
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func loadConfig(path string) *DDNSConfig {
	var ddns DDNSConfig
	//加载toml文件
	if _, err := toml.DecodeFile(path, &ddns); err != nil {
		log.Println("decode config file err")
	}
	return &ddns
}

func loadConfigs() {
	for _, v := range CONFIG_PATH {
		files, _ := ioutil.ReadDir(v)
		for _, f := range files {
			filename := f.Name()
			filesuffix := path.Ext(filename)
			if f.IsDir() || filesuffix != ".toml" {
				continue
			}

			configpath := path.Join(v, f.Name())
			log.Printf("加载配置文件：%s", filename)
			CONFIG_MAP[filename[0:len(filename)-len(filesuffix)]] = loadConfig(configpath)
		}
	}
}

func getClient(aliAccess *AliAccess) *alidns.Client {
	client, _ := alidns.NewClientWithAccessKey(aliAccess.Region, aliAccess.AccessId, aliAccess.AccessKey)
	return client
}

func getRecord(rr string, ddnsConfig *DDNSConfig, client *alidns.Client) alidns.Record {

	request := alidns.CreateDescribeDomainRecordsRequest()
	request.Scheme = "https"
	request.DomainName = ddnsConfig.AliAccess.Domain
	response, err := client.DescribeDomainRecords(request)
	if err != nil {
		log.Println(err.Error())
	}
	// 过滤符合条件的子域名信息。
	result := linq.From(response.DomainRecords.Record).Where(func(c interface{}) bool {
		return c.(alidns.Record).RR == rr
	}).First()

	return result.(alidns.Record)

}

func updateDomain(ddnsConfig *DDNSConfig, domain *Domain, client *alidns.Client) {
	record := getRecord(domain.Rr, ddnsConfig, client)
	ip := domain.Value
	if domain.UpdateType == "network" {
		ip = getIP()
	}
	if ip == record.Value {
		log.Printf("[%s]IP一致，无需更新 => [%s]", domain.Rr, ip)
		return
	}
	request := alidns.CreateUpdateDomainRecordRequest()
	request.Scheme = "https"
	request.RecordId = record.RecordId
	request.RR = domain.Rr
	request.Type = domain.DomainType
	request.Value = ip
	log.Printf("更新[%s]: %s => %s", domain.Rr, record.Value, ip)
	_, err := client.UpdateDomainRecord(request)
	if err != nil {
		log.Print(err.Error())
	}
}

func update() {
	for key, ddnsConfig := range CONFIG_MAP {
		ddnsConfig.AliAccess.Domain = key
		// 获取DDNS客户端
		client := getClient(ddnsConfig.AliAccess)

		for _, domain := range *ddnsConfig.Domains {
			log.Println(domain)
			updateDomain(ddnsConfig, &domain, client)
		}
	}
}

var run = flag.Uint64("r", 30, "定时执行，30分钟执行一次")
var one = flag.Bool("o", true, "单次执行, 默认为false")

func main() {
	loadConfigs()
	flag.Parse()
	log.Println(*one)
	if *one {
		log.Println("执行一次.............")
		update()
		log.Println("结束执行.............")
	} else {
		s := gocron.NewScheduler()
		s.Every(*run).Minutes().DoSafely(update)
		<-s.Start()
	}

}
