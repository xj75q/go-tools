package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	pathMark = string(os.PathSeparator)
)

type fieldName struct {
	inputPath    string
	outputPath   string
	oldName      string
	newName      string
	sameFileName string
	nameFormPath bool
	addStr       string
	nameLoc      bool
	subStr       string
	subLoc       string
	fileInfoChan chan interface{}
	layerChan    chan interface{}
	newFileChan  chan string
}

func NewHandler() *fieldName {
	return &fieldName{
		fileInfoChan: make(chan interface{}, 20),
		layerChan:    make(chan interface{}, 20),
		newFileChan:  make(chan string, 20),
	}
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func IsFile(path string) bool {
	return !IsDir(path)
}

func (f *fieldName) inputFileInfo(ctx *cli.Context) error {
	err := filepath.Walk(f.inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fpath := filepath.Clean(fmt.Sprintf("%s%s%s", f.inputPath, pathMark, info.Name()))
		outputpath := filepath.Clean(f.outputPath)
		if info.IsDir() && fpath == outputpath {
			return nil
		}
		if IsFile(path) {
			fileInfo := make(map[string]interface{})
			fileInfo[path] = info
			go func(ctx *cli.Context) {
				f.fileInfoChan <- fileInfo
			}(ctx)
		}
		return nil
	})
	return err
}

func (f *fieldName) usePathName(ctx *cli.Context) error {
	if f.nameFormPath == false {
		return nil
	}

	count := 0
	var layerList = []string{}
	for {
		select {
		case chanInfo := <-f.fileInfoChan:
			finfo := chanInfo.(map[string]interface{})
			for path, value := range finfo {
				Info := value.(os.FileInfo)
				pathInfo := strings.Split(path, pathMark)
				pathName := strings.Join(pathInfo[len(pathInfo)-2:len(pathInfo)-1], "")
				fpath := strings.Join(pathInfo[:len(pathInfo)-1], pathMark)

				if strings.Compare(fpath, filepath.Clean(f.inputPath)) == 1 {
					layerList = append(layerList, path)
				} else {
					count++
					newfile := fmt.Sprintf("%s%s%s-%v.%s", fpath, pathMark, pathName, count, strings.Split(Info.Name(), ".")[1])
					err := os.Rename(path, newfile)
					if err != nil {
						return err
					}
					fmt.Printf("重命名为 %s 成功，请查看...\n", newfile)
				}

			}
		default:
			f.sortLayer(ctx, layerList)
			return nil
		}
	}
}

func (f *fieldName) sortLayer(ctx *cli.Context, layerList []string) {
	keys := []string{}
	for _, layer := range layerList {
		pathInfo := strings.Split(layer, pathMark)
		key := strings.Join(pathInfo[:len(pathInfo)-1], pathMark)
		keys = append(keys, key)
	}

	sort.Strings(keys)
	result := make(map[string]int)
	for _, item := range keys {
		result[item]++
	}

	var numList = []map[string]int{}
	for key, value := range result {
		var vList = make([]int, value)
		for index := range vList {
			var fileNum = make(map[string]int)
			fileNum[key] = index + 1
			numList = append(numList, fileNum)
		}
	}
	f.generateFileNum(ctx, layerList, numList)
}

func (f *fieldName) generateFileNum(ctx *cli.Context, layerList []string, numList []map[string]int) {
	var newfile string
	go func() {
		for _, layer := range layerList {
			pathInfo := strings.Split(layer, pathMark)
			key := strings.Join(pathInfo[:len(pathInfo)-1], pathMark)
			fileType := strings.Split(layer, ".")
			fname := strings.Join(pathInfo[len(pathInfo)-2:len(pathInfo)-1], pathMark)
			ftype := strings.Join(fileType[len(fileType)-1:], "")

			for _, fileNum := range numList {
				if fileNum[key] != 0 {
					if f.nameFormPath == true {
						newfile = fmt.Sprintf("%s%s%s-%v.%s", key, pathMark, fname, fileNum[key], ftype)
						f.newFileChan <- newfile
					} else {
						newfile = fmt.Sprintf("%s%s%s-%v.%s", key, pathMark, f.sameFileName, fileNum[key], ftype)
						f.newFileChan <- newfile
					}
					delete(fileNum, key)
				}
			}
		}
	}()

	f.manageLayer(layerList)

}

func (f *fieldName) manageLayer(layerList []string) {
	for _, layer := range layerList {
		select {
		case newfile := <-f.newFileChan:
			err := os.Rename(layer, newfile)
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Printf("重命名为 %s 成功，请查看...\n", newfile)
		}
	}
}

func (f *fieldName) useSameName(ctx *cli.Context) error {
	count := 0
	var layerList = []string{}
	for {
		select {
		case chanInfo := <-f.fileInfoChan:
			finfo := chanInfo.(map[string]interface{})
			for path, value := range finfo {
				Info := value.(os.FileInfo)
				pathInfo := strings.Split(path, pathMark)
				fpath := strings.Join(pathInfo[:len(pathInfo)-1], pathMark)
				if strings.Compare(fpath, filepath.Clean(f.inputPath)) == 1 {
					layerList = append(layerList, path)
				} else {
					count++
					newfile := fmt.Sprintf("%s%s%s-%v.%s", fpath, pathMark, f.sameFileName, count, strings.Split(Info.Name(), ".")[1])

					err := os.Rename(path, newfile)
					if err != nil {
						return err
					}
					fmt.Printf("重命名为 %s 成功，请查看...\n", newfile)
				}

			}
		default:
			f.sortLayer(ctx, layerList)
			return nil
		}
	}
}

func (f *fieldName) uniquefname(array []string) []string {
	//result := []map[string]int{}
	result := []string{}
	seen := map[string]bool{}
	for _, value := range array {
		if _, ok := seen[value]; !ok {
			//result = append(result, map[string]int{value: 0})
			result = append(result, value)
			seen[value] = true
		}
	}

	return result
}

func (f *fieldName) replaceFileName(ctx *cli.Context) error {
	for {
		select {
		case chanInfo := <-f.fileInfoChan:
			finfo := chanInfo.(map[string]interface{})
			for path, value := range finfo {
				Info := value.(os.FileInfo)
				infoName := strings.Split(Info.Name(), ".")
				pathInfo := strings.Split(path, pathMark)
				fpath := strings.Join(pathInfo[:len(pathInfo)-1], pathMark)
				newname := strings.ReplaceAll(infoName[0], f.oldName, f.newName)
				newfile := fmt.Sprintf("%s%s%s.%s", fpath, pathMark, newname, infoName[1])

				err := os.Rename(path, newfile)
				if err != nil {
					return err
				}
				fmt.Printf("重命名为 %s 成功，请查看...\n", newfile)

			}

		default:
			return nil
		}
	}

}

func (f *fieldName) addFileName(ctx *cli.Context) error {
	var (
		newName string
		newfile string
	)

	for {
		select {
		case chanInfo := <-f.fileInfoChan:
			finfo := chanInfo.(map[string]interface{})
			for path, value := range finfo {
				Info := value.(os.FileInfo)
				infoName := strings.Split(Info.Name(), ".")
				sList := strings.Split(path, pathMark)
				originFilePath := strings.Join(sList[:len(sList)-1], pathMark)
				if f.nameLoc {
					newName = fmt.Sprintf("%v-%v.%s", f.addStr, infoName[0], infoName[1])

				} else {
					newName = fmt.Sprintf("%v-%v.%s", infoName[0], f.addStr, infoName[1])
				}
				outPath := filepath.Clean(f.outputPath)
				if outPath == "." {
					newfile = originFilePath + pathMark + newName
				} else {
					newfile = outPath + pathMark + newName
				}

				//fmt.Println(path, newfile)

				err := os.Rename(path, newfile)
				if err != nil {
					return err
				}
				fmt.Printf("重命名为 %s 成功，请查看...\n", newfile)

			}

		default:
			return nil

		}
	}
}

func (f *fieldName) subFileName(ctx *cli.Context) {
	for {
		select {
		case chanInfo := <-f.fileInfoChan:
			var newName string
			finfo := chanInfo.(map[string]interface{})
			for path, value := range finfo {
				info := value.(os.FileInfo)
				finfo := strings.Split(info.Name(), ".")
				if f.subStr == "" || !strings.Contains(finfo[0], f.subStr) {
					return
				}

				pathInfo := strings.Split(path, pathMark)
				fpath := strings.Join(pathInfo[:len(pathInfo)-1], pathMark)
				strIndex := strings.Index(finfo[0], f.subStr)
				switch {
				case f.subLoc == "all":
					newName = strings.ReplaceAll(finfo[0], f.subStr, "")
					newfile := fmt.Sprintf("%s%s%s.%s", fpath, pathMark, newName, finfo[1])

					//fmt.Println("*****", path, newfile)

					err := os.Rename(path, newfile)
					if err != nil {
						fmt.Println(err.Error())
						return
					}
					fmt.Printf("重命名为 %s 成功，请查看...\n", newfile)
				case f.subLoc == "left":
					if strIndex == 0 {
						newName = strings.TrimLeft(finfo[0], f.subStr)
						newfile := fmt.Sprintf("%s%s%s.%s", fpath, pathMark, newName, finfo[1])

						//fmt.Println("*****", path, newfile)

						err := os.Rename(path, newfile)
						if err != nil {
							fmt.Println(err.Error())
							return
						}
						fmt.Printf("重命名为 %s 成功，请查看...\n", newfile)
					}

				case f.subLoc == "right":
					if strIndex == len([]rune(finfo[0]))-1 {
						newName = strings.TrimRight(finfo[0], f.subStr)
						newfile := fmt.Sprintf("%s%s%s.%s", fpath, pathMark, newName, finfo[1])

						//fmt.Println("*****", path, newfile)

						err := os.Rename(path, newfile)
						if err != nil {
							fmt.Println(err.Error())
							return
						}
						fmt.Printf("重命名为 %s 成功，请查看...\n", newfile)
					}

				default:
					return

				}

			}

		}
	}
}

var (
	handler = NewHandler()
	authors = []*cli.Author{
		{
			Name: "developed by qxz",
		},
	}
	cliFlags = []cli.Flag{
		&cli.StringFlag{
			Name:     "input",
			Aliases:  []string{"i"},
			Usage:    "操作文件路径（必填）",
			Value:    "./",
			Required: false,
		},

		&cli.StringFlag{
			Name:     "output",
			Aliases:  []string{"o"},
			Usage:    "输出文件的路径（选填）",
			Required: false,
		},
	}

	cliCommands = []*cli.Command{
		{
			Name:    "usepathname",
			Aliases: []string{"use"},
			Usage:   "使用文件名称作为新文件的名字",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "frompath",
					Aliases:  []string{"p"},
					Usage:    "是否使用文件夹名字，默认为false，如需启用设为true",
					Value:    false,
					Required: false,
				},
			},
			Action: func(ctx *cli.Context) error {
				handler.nameFormPath = ctx.Bool("p")
				handler.outputPath = ctx.String("output")
				inPath := ctx.String("input")
				if inPath == "./" {
					handler.inputPath, _ = os.Getwd()
				} else {
					handler.inputPath = inPath
				}
				handler.inputFileInfo(ctx)
				time.Sleep(500 * time.Millisecond)
				handler.usePathName(ctx)
				return nil
			},
		},
		{
			Name:    "samename",
			Aliases: []string{"same"},
			Usage:   "使用相同名字作为文件名",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Aliases:  []string{"n"},
					Usage:    "请输入要使用的名字",
					Required: false,
				},
			},
			Action: func(ctx *cli.Context) error {
				sameName := ctx.String("name")
				if sameName == "" {
					fmt.Println("使用的新名字不能为空，请填写正确的文件名")
					os.Exit(0)
				} else {
					handler.sameFileName = sameName
				}
				handler.outputPath = ctx.String("output")
				inPath := ctx.String("input")
				if inPath == "./" {
					handler.inputPath, _ = os.Getwd()
				} else {
					handler.inputPath = inPath
				}
				handler.inputFileInfo(ctx)
				time.Sleep(500 * time.Millisecond)

				handler.useSameName(ctx)
				return nil
			},
		},
		{

			Name:      "replace",
			Aliases:   []string{"rep"},
			Usage:     "替换文件的字符串",
			UsageText: "",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "oldname",
					Aliases:  []string{"o"},
					Usage:    "请填写需替换的字符串",
					Value:    "new",
					Required: false,
				},

				&cli.StringFlag{
					Name:     "newname",
					Aliases:  []string{"n"},
					Usage:    "请填写替换后的字符串",
					Required: false,
				},
			},
			Action: func(ctx *cli.Context) error {
				handler.oldName = ctx.String("oldname")
				handler.newName = ctx.String("newname")
				handler.outputPath = ctx.String("output")

				inPath := ctx.String("input")

				if inPath == "./" {
					handler.inputPath, _ = os.Getwd()
				} else {
					handler.inputPath = inPath
				}
				handler.inputFileInfo(ctx)
				time.Sleep(500 * time.Millisecond)
				handler.replaceFileName(ctx)
				return nil
			},
		}, {
			Name:    "addsign",
			Aliases: []string{"add"},
			Usage:   "默认使用new，如果需要修改为其他名称，请填写",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "addstr",
					Aliases:  []string{"n"},
					Usage:    "默认使用new，如果需要修改为其他名称，请填写",
					Value:    "new",
					Required: false,
				},

				&cli.BoolFlag{
					Name:     "signloc",
					Aliases:  []string{"l"},
					Usage:    "重命名标志位置在左侧，默认true，如需更改使用 -l=false ",
					Value:    true,
					Required: false,
				},
			},
			Action: func(ctx *cli.Context) error {
				handler.addStr = ctx.String("addstr")
				handler.nameLoc = ctx.Bool("signloc")
				handler.outputPath = ctx.String("output")
				inPath := ctx.String("input")
				if inPath == "./" {
					handler.inputPath, _ = os.Getwd()
				} else {
					handler.inputPath = inPath
				}
				handler.inputFileInfo(ctx)
				time.Sleep(500 * time.Millisecond)
				handler.addFileName(ctx)
				return nil
			},
		}, {
			Name:    "substr",
			Aliases: []string{"sub"},
			Usage:   "在文件名中删除某个字符",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "subname",
					Aliases:  []string{"n"},
					Usage:    "请填写要删除的字符串",
					Required: false,
				},
				&cli.StringFlag{
					Name:    "subloc",
					Aliases: []string{"l"},
					Usage:   "请填写要删除字符串的位置[默认为全部替换如需更改请使用：left-替换左侧字符，right-替换右侧字符]",

					Value:    "all",
					Required: false,
				},
			},
			Action: func(ctx *cli.Context) error {
				handler.subStr = ctx.String("subname")
				handler.outputPath = ctx.String("output")
				handler.subLoc = ctx.String("subloc")
				inPath := ctx.String("input")
				if inPath == "./" {
					handler.inputPath, _ = os.Getwd()
				} else {
					handler.inputPath = inPath
				}

				handler.inputFileInfo(ctx)
				time.Sleep(500 * time.Millisecond)
				handler.subFileName(ctx)

				return nil
			},
		},
	}
)

func main() {
	defer close(handler.fileInfoChan)
	app := cli.NewApp()
	app.Name = "【文件批量重命名】"
	app.Usage = "input the file path and auto switch to new file"
	app.Description = ""
	app.Flags = cliFlags
	app.Commands = cliCommands
	app.Authors = authors
	app.Action = func(ctx *cli.Context) error {
		return nil
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(">> 出错了！！！:", err)
		os.Exit(0)
	}
}