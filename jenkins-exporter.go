package main
import (
	"log"
	"flag"
	"io/ioutil"
	"encoding/json"
	"regexp"
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"strconv"
	"time"
)

type JenkinsJobs struct {
	Class string `json:"_class"`
	Jobs  []struct {
		Class string `json:"_class"`
		Name  string `json:"name"`
	} `json:"jobs"`
}

var (
	jenkins_builds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "jenkins",
			Help: "Number of Builds",
		},
		[]string{"Program"},
	)
)

func init() {
	prometheus.MustRegister(jenkins_builds)
}

func getFromJenkinsApi(url string,Url string, Username string, Password string) {
retry:
	client := &http.Client{}
    req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(Username, Password)
    resp, err := client.Do(req)
    if err != nil{
		log.Printf("%s",err)
		time.Sleep(20 * time.Second)
		goto retry 
    }
    bodyText, err := ioutil.ReadAll(resp.Body)

	var jobs JenkinsJobs
	json.Unmarshal([]byte(bodyText), &jobs)

	r, _ := regexp.Compile("(.*(deploy|toggle|force))")
	for _, job := range jobs.Jobs  {
		if (r.MatchString(job.Name)) {
			url := Url+"/job/"+job.Name+"/api/xml?xpath=/*/lastStableBuild/number"
			count := BuildCount(url, Username, Password)
			jenkins_builds.With(prometheus.Labels{"Program":job.Name}).Set(count)
			}		
	}	
}

func BuildCount(url string, username string, password string, ) float64 {
	var  MetricValue = 0.1
repeat:
	client := &http.Client{}
    req, err := http.NewRequest("GET", url, nil)
    req.SetBasicAuth(username, password)
    resp, err := client.Do(req)
    if err != nil{
		log.Printf("%s",err)
		time.Sleep(20 * time.Second)
		goto repeat 
    }
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil{
        
        log.Printf("%s",err)
    }
	r, _ := regexp.Compile("[0-9]+")
	re:= r.FindAllString(string(bodyText), -1)
	if len(re) != 0 {
		MetricValue, _ := strconv.ParseFloat(re[0],64)
		return MetricValue	
		} 
	return MetricValue
}

func recordMetrics(JenkinsUrl string, url string,username string, password string) {
	go func() {
			for {
				getFromJenkinsApi(JenkinsUrl, url, username, password)
				time.Sleep(2 * time.Second)

			}
	}()
}

func main() {
	port:=flag.Int( "port", 2115, "Port Number to listen")
	url := flag.String("url", "" , "Jenkins Url")
	username := flag.String("user", "" , "Jenkins username")
	password := flag.String("pass", "" , "Jenkins password")
	flag.Parse()
	Port := ":"+strconv.Itoa(*port)
	JenkinsUrl := *url + "/api/json?tree=jobs[name]"
	recordMetrics(JenkinsUrl, *url, *username, *password)
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(Port, nil)
}


