package downloader

import (
	"bytes"
	"fmt"
	"github.com/zzbkszd/godown/common"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

/**
m3u8元数据
*/
type M3u8MetaData struct {
	Version        int          // 版本
	PlayListType   string       // 列表类别
	IsTsList       bool         // 是否是ts列表文件，或者是播放列表文件
	TsList         []M3u8TsLink // ts
	EncryptMethod  string       // 加密方法
	EncryptKeyUrl  string       // 加密key的url
	EncryptKey     string       // 加密key
	EncryptIv      string       // 加密iv
	OriginData     string       // 源数据
	TargetDuration int
	MediaSequence  int
}

type M3u8TsLink struct {
	ExtInf   string
	Url      string
	FileName string
}

func ParseM3u8(content string) *M3u8MetaData {
	baseList := strings.Split(content, "\n")
	meta := &M3u8MetaData{
		OriginData:    content,
		TsList:        make([]M3u8TsLink, 0),
		IsTsList:      true,
		EncryptMethod: "NONE",
		EncryptKeyUrl: "",
		EncryptIv:     "",
	}
	privExtInf := ""
	for _, line := range baseList {
		line = strings.Trim(line, " ")
		if strings.HasSuffix(line, "\r") {
			line = line[:len(line)-1]
		}
		if len(line) == 0 { // 空行跳过
			continue
		} else if strings.HasPrefix(line, "#") {
			t := strings.Index(line, ":")
			if t == -1 {
				t = len(line)
			}
			field := line[1:t]
			value := ""
			if t < len(line) {
				value = line[t+1:]
			}
			switch field {
			case "EXT-X-VERSION":
				meta.Version, _ = strconv.Atoi(value)
			case "EXT-X-PLAYLIST-TYPE":
				meta.PlayListType = value
			case "EXT-X-KEY":
				encrypt := make(map[string]string)
				kvs := strings.Split(value, ",")
				for _, kv := range kvs {
					eqidx := strings.Index(kv, "=")
					encrypt[kv[:eqidx]] = kv[eqidx+1:]
				}
				if method, ok := encrypt["METHOD"]; ok {
					meta.EncryptMethod = method
				}
				if uri, ok := encrypt["URI"]; ok {
					if uri[0] == '"' {
						uri = uri[1 : len(uri)-1]
					}
					meta.EncryptKeyUrl = uri
				}
				if iv, ok := encrypt["IV"]; ok {
					meta.EncryptIv = iv
				}
			case "EXTINF":
				privExtInf = line
			case "EXT-X-TARGETDURATION":
				meta.TargetDuration, _ = strconv.Atoi(value)
			case "EXT-X-MEDIA-SEQUENCE":
				meta.MediaSequence, _ = strconv.Atoi(value)
			}
		} else {
			meta.TsList = append(meta.TsList, M3u8TsLink{ExtInf: privExtInf, Url: line, FileName: GetUrlFileName(line)})
		}
	}
	return meta
}

func (d M3u8MetaData) WriteHeader(out *bytes.Buffer, keystore string) {
	out.WriteString(fmt.Sprintln("#EXTM3U"))
	out.WriteString(fmt.Sprintf("#EXT-X-VERSION:%d\n", d.Version))
	out.WriteString(fmt.Sprintf("#EXT-X-PLAYLIST-TYPE:%s\n", d.PlayListType))
	out.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION::%d\n", d.TargetDuration))
	out.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", d.MediaSequence))
	out.WriteString(fmt.Sprintln("#EXT-X-INDEPENDENT-SEGMENTS"))
	if d.EncryptMethod != "NONE" {
		encrypt := fmt.Sprintf("METHOD=%s,URI=%s", d.EncryptMethod, keystore)
		if d.EncryptIv != "" {
			encrypt += ",IV=" + d.EncryptIv
		}
		out.WriteString(fmt.Sprintf("#EXT-X-KEY:%s\n", encrypt))
	}
}

/**
m3u8 下载器
暂不支持加密格式，未进行格式转换
支持多线程并发下载，默认线程数为5
todo 已知设计BUG： 当因网络链接之类的问题导致下载确实无法进行时会无限次数重试。

20200809 改变设计思路：
下载m3u8文件，下载ts文件，下载秘钥，重新生成m3u8，最后用ffmpeg来处理或不处理直接使用
*/
type M3u8Downloader struct {
	AbstractDownloader
	Threads int
	Header  http.Header
}

func (d *M3u8Downloader) Download(urlstr, dist string) (string, error) {
	d.Init()
	if d.Threads == 0 {
		d.Threads = 5
	}
	d.PrepareDist(dist)
	tsdir, err := ioutil.TempDir(path.Dir(dist), "ts*")
	if err != nil {
		return "", err
	}
	m3u8File, err := d.FetchText(QuickRequest(http.MethodGet, urlstr, d.Header))
	if err != nil {
		return "", err
	}
	metadata := ParseM3u8(m3u8File)
	if metadata.EncryptMethod != "NONE" {
		key, err := d.FetchText(QuickRequest(http.MethodGet, metadata.EncryptKeyUrl, nil))
		if err != nil {
			return "", err
		}
		metadata.EncryptKey = key
	}
	//d.doDownload(metadata, urlstr, tsdir)
	err = d.createIndexFile(metadata, dist, tsdir)
	//err = d.combinTs(metadata, dist, tsdir)
	if err != nil {
		return "", err
	}
	return dist, nil
}

func (d *M3u8Downloader) createIndexFile(meta *M3u8MetaData, dist, tsdir string) error {
	newM3u8 := new(bytes.Buffer)
	distDir, distFile := path.Split(dist)
	if meta.EncryptMethod != "NONE" {
		ioutil.WriteFile(path.Join(distDir, distFile+".key"), []byte(meta.EncryptKey), 0777)
	}
	relateedDir := tsdir[len(distDir):] // 文件相对m3u8的相对路径，其实也就是取tsdir的文件夹名称
	meta.WriteHeader(newM3u8, distFile+".key")
	for _, ts := range meta.TsList {
		tsPath := path.Join(relateedDir, ts.FileName)
		newM3u8.WriteString(fmt.Sprintln(ts.ExtInf))
		newM3u8.WriteString(fmt.Sprintf("%s\r\n", tsPath))
	}
	//fmt.Println(string(newM3u8.Bytes()))
	ioutil.WriteFile(dist, newM3u8.Bytes(), 0777)
	return nil

}

func (d *M3u8Downloader) combinTs(meta *M3u8MetaData, dist, tsdir string) error {
	fmt.Printf("[M3U8 Downloader] start combin ts data \n")
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	defer distFile.Close()
	if e != nil {
		panic(e)
	}
	for _, ts := range meta.TsList {
		tsPath := path.Join(tsdir, ts.FileName)
		tsFile, e := os.OpenFile(tsPath, os.O_RDONLY, 0777)
		if e != nil {
			panic(e)
		}
		_, err := io.Copy(distFile, tsFile)
		if err != nil {
			panic(err)
		}
		tsFile.Close()
		os.Remove(tsPath)
	}
	finfo, err := distFile.Stat()
	if err != nil {
		return err
	}
	fileLength := finfo.Size()
	if fileLength < 1024*1024 {
		return fmt.Errorf("file size too small: %s", dist)
	}
	os.Remove(tsdir)
	return nil
}

func (d *M3u8Downloader) doDownload(meta *M3u8MetaData, baseUrl, tsdir string) {
	parent := strings.Split(baseUrl, "/")
	base := strings.Join(parent[:len(parent)-1], "/")
	d.InitProgress(int64(len(meta.TsList)), false)
	defer d.CloseProgress()
	taskSet := make([]func() error, 0)
	for _, ts := range meta.TsList {
		keyUrl := ts.Url
		taskSet = append(taskSet, func() error {
			tsUrl := strings.Join([]string{base, keyUrl}, "/")
			if strings.HasPrefix(keyUrl, "http") {
				tsUrl = keyUrl
			}
			tsDist := path.Join(tsdir, GetUrlFileName(keyUrl))
			err := d.HttpDown(QuickRequest(http.MethodGet, tsUrl, nil), tsDist)
			if err != nil {
				return err
			}
			d.UpdateProgress(1)
			return nil
		})
	}
	cycle := common.MultiTaskCycle{
		Threads:   d.Threads,
		TryOnFail: true,
	}
	cycle.CostTasks(taskSet)
}
