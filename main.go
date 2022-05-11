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
	alidns "github.com/alibabacloud-go/alidns-20150109/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/jasonlvhit/gocron"
)

var ConfigPath = []string{"~/.config/aliddns", "./configs"}

var ConfigMap = make(map[string]*DDNSConfig)

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
	responseClient, err := http.Get("https://ipv4.ipw.cn/api/ip/myip")

	if err != nil {
		log.Printf("获取外网 IP 失败，请检查网络\n")
		log.Panic(err)
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
	for _, v := range ConfigPath {
		files, _ := ioutil.ReadDir(v)
		for _, f := range files {
			filename := f.Name()
			filesuffix := path.Ext(filename)
			if f.IsDir() || filesuffix != ".toml" {
				continue
			}

			configpath := path.Join(v, f.Name())
			log.Printf("加载配置文件：%s", filename)
			ConfigMap[filename[0:len(filename)-len(filesuffix)]] = loadConfig(configpath)
		}
	}
}

func getClient(aliAccess *AliAccess) (client *alidns.Client, _err error) {
	config := &openapi.Config{
		// 您的AccessKey ID
		AccessKeyId: &aliAccess.AccessId,
		// 您的AccessKey Secret
		AccessKeySecret: &aliAccess.AccessKey,
		// 设定协议 HTTPS/HTTP
		Protocol: tea.String("HTTPS"),
	}
	// 访问的域名
	config.Endpoint = tea.String("alidns.cn-hangzhou.aliyuncs.com")
	client = &alidns.Client{}
	client, _err = alidns.NewClient(config)
	return client, _err
}

func getRecord(rr *string, ddnsConfig *DDNSConfig, client *alidns.Client) *alidns.DescribeDomainRecordsResponseBodyDomainRecordsRecord {

	request := alidns.DescribeDomainRecordsRequest{Lang: tea.String("g")}
	request.DomainName = &ddnsConfig.AliAccess.Domain
	response, err := client.DescribeDomainRecords(&request)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	if response == nil {
		return nil
	}

	// 过滤符合条件的子域名信息。
	result := linq.From(response.Body.DomainRecords.Record).Where(func(c interface{}) bool {
		return *c.(*alidns.DescribeDomainRecordsResponseBodyDomainRecordsRecord).RR == *rr
	}).First()

	if result == nil {
		return nil
	}

	return result.(*alidns.DescribeDomainRecordsResponseBodyDomainRecordsRecord)

}

func updateDomain(ddnsConfig *DDNSConfig, domain *Domain, client *alidns.Client) {
	record := getRecord(&domain.Rr, ddnsConfig, client)
	ip := domain.Value
	if domain.UpdateType == "network" {
		ip = getIP()
	}
	if ip == *record.Value {
		log.Printf("[%s]IP一致，无需更新 => [%s]", domain.Rr, ip)
		return
	}
	request := &alidns.UpdateDomainRecordRequest{Lang: tea.String("g")}
	request.RecordId = record.RecordId
	request.RR = &domain.Rr
	request.Type = &domain.DomainType
	request.Value = &ip
	log.Printf("更新[%s]: %s => %s", domain.Rr, *record.Value, ip)
	_, err := client.UpdateDomainRecord(request)
	if err != nil {
		log.Print(err.Error())
	}
}

func update() {
	for key, ddnsConfig := range ConfigMap {
		ddnsConfig.AliAccess.Domain = key
		// 获取DDNS客户端
		client, _ := getClient(ddnsConfig.AliAccess)

		for _, domain := range *ddnsConfig.Domains {
			updateDomain(ddnsConfig, &domain, client)
		}
	}
}

var run = flag.Uint64("e", 30, "定时执行，默认30分钟执行一次")
var one = flag.Bool("o", true, "单次执行, 默认为false")

func main() {
	loadConfigs()
	flag.Parse()
	if *one {
		log.Println("执行一次.............")
		update()
		log.Println("结束执行.............")
	} else {
		log.Printf("定时执行，%d分钟一次.............", *run)
		update()
		s := gocron.NewScheduler()
		s.Every(*run).Minutes().Do(update)
		<-s.Start()
	}

}
