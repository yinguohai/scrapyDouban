package douban

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"regexp"
	"spider/request"
	"strconv"
	"strings"
	"gotools"
)

type douban struct {
	spider      request.Spider
	templateUrl string
	titles      []string
	redisKey  	string
}

var dbspider douban

/**
	包的初始化函数，每个包都可以有多个init函数，
 */
func init() {
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.81 Safari/537.36",
		"Host":       "movie.douban.com",
	}
	dbspider = douban{
		spider: request.Spider{
			Header: headers,
		},
		templateUrl: "https://movie.douban.com/top250?start=%v&filter=",
		titles:      []string{"href", "name", "img", "name2","score","comment","page"},
		redisKey: "douban",
	}
}

/**
	存储数据
 */
func (dBan *douban) redisStore(content string) error {
	client := redis.NewClient(&redis.Options{
		Addr:"127.0.0.1:6379",
		Password:"123456",
		DB:0,
	})

	_, err := client.Ping().Result()

	if err != nil {
		return fmt.Errorf("链接出错！！！%v",nil)
	}

	fmt.Println(dbspider.redisKey)
	client.LPush(dbspider.redisKey,content)

	return nil
}

func (dBan *douban) parse(content *[]uint8, page int) (*[][]string, error) {
	data := string(*content)
	startPosition := strings.Index(data, "class=\"grid_view\"")

	endPosition := strings.Index(data, "class=\"paginator\"")

	olContent := data[startPosition:endPosition]

	match := regexp.MustCompile(`(?mUs)class="pic".*href="(.*)".*img.*alt="(.*").*src="(.*)".*</div>.*class="title">(.*)</span>.*property="v:average">(.*)</span>.*</span>.*<span>(.*)</span>`)

	result := match.FindAllStringSubmatch(olContent, -1)

	for _, v := range result {
		v := append(v,strconv.Itoa(page))
		data , _ := gotools.ListToMap(dBan.titles,v[1:])
		item,_ := json.Marshal(data)
		err := dBan.redisStore(string(item))
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	return &result, nil
}

func (dBan *douban) scrapy(page int, single chan bool) {
	href := fmt.Sprintf(dbspider.templateUrl, (page-1)*25)

	content, err := dbspider.spider.Get(href)

	if err != nil {
		panic(err.Error())
	}

	dBan.parse(content, page)

	single <- true
}

func Run() {
	signle := make(chan bool, 5)
	for i := 1; i <= 10; i++ {
		//开启10个协程同时去爬豆瓣的数据
		go dbspider.scrapy(i, signle)
	}

	gotools.IsDoneChan(signle, 10, 0)

	fmt.Println("爬取结束")
}
