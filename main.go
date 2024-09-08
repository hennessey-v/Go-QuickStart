package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const configFile = "config.json"

// Config 结构体用于存储配置信息
type Config struct {
	ProjectDir string   `json:"projectDir"`
	SubDir     []string `json:"subDir"`
	Remarks    []struct {
		Name   string `json:"name"`
		Remark string `json:"remark"`
	} `json:"remarks"`
}

func main() {
	config, err := readConfig()
	if err != nil {
		fmt.Println("无法读取配置文件:", err)
		return
	}

	if err := runProjectMenu(config.ProjectDir, config.SubDir, config.Remarks); err != nil {
		fmt.Println("程序异常:", err)
	}
}

func runProjectMenu(projectDir string, subDirs []string, remarks []struct {
	Name   string `json:"name"`
	Remark string `json:"remark"`
}) error {
	// 读取项目目录下的文件夹列表
	folders, err := listFolders(projectDir, subDirs)
	if err != nil {
		return fmt.Errorf("无法读取文件夹: %v", err)
	}
	// 切换到项目目录
	if err := os.Chdir(projectDir); err != nil {
		return err
	}

	// 循环显示文件夹列表，直到用户选择成功或者主动退出
	for {
		printFolderList(folders, subDirs, remarks)
		choice, err := getUserChoice(len(folders))
		if err != nil {
			fmt.Println(err)
			continue
		}
		selectedFolder := folders[choice-1].Name()
		if err := runCommand(selectedFolder, subDirs, remarks); err != nil {
			return fmt.Errorf("无法执行命令: %v", err)
		}
		break
	}

	return nil
}

func readConfig() (*Config, error) {
	// 检测配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// 如果配置文件不存在，则创建一个默认的配置文件,路径为程序所在目录
		exePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("无法获取当前执行文件的路径: %v", err)
		}
		exeDir := filepath.Dir(exePath)

		defaultConfig := &Config{
			ProjectDir: exeDir,
			SubDir:     nil, // 默认为空
		}
		// 创建并写入配置文件
		if err := writeConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("无法创建配置文件: %v", err)
		}
		return defaultConfig, nil
	}

	// 读取配置文件
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 解析配置文件内容到 Config 结构体
	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func writeConfig(config *Config) error {
	// 创建配置文件
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// 编码配置信息并写入配置文件
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// 获取指定目录下的文件夹列表，将子目录置顶
func listFolders(dir string, subDirs []string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var (
		folders     []os.DirEntry
		subDirNames = make(map[string]bool)
	)

	// 将子目录名称存入 map 中，便于后续检索
	for _, subDir := range subDirs {
		subDirNames[subDir] = true
	}

	// 首先将子目录加入 folders
	for _, entry := range entries {
		if entry.IsDir() && subDirNames[entry.Name()] {
			folders = append(folders, entry)
		}
	}

	// 将除子目录外的其他文件夹加入 folders
	for _, entry := range entries {
		if entry.IsDir() && !subDirNames[entry.Name()] {
			folders = append(folders, entry)
		}
	}

	return folders, nil
}

// 打印文件夹列表，如果是子目录，添加*号标记，如果有备注，显示备注
func printFolderList(folders []os.DirEntry, subDirs []string, remarks []struct {
	Name   string `json:"name"`
	Remark string `json:"remark"`
}) {
	fmt.Println("启动项目：")
	for i, folder := range folders {
		folderName := folder.Name()
		remark := ""
		for _, r := range remarks {
			if r.Name == folderName {
				remark = fmt.Sprintf("  [%s]", r.Remark)
				break
			}
		}
		if contains(folderName, subDirs) {
			folderName += "*"
		}
		fmt.Printf("%d. %s%s\n", i+1, folderName, remark)
	}
}

// 获取用户选择的文件夹编号
func getUserChoice(maxChoice int) (int, error) {
	var choice int
	fmt.Print("请输入要运行的文件夹编号: ")
	_, err := fmt.Scanln(&choice)
	if err != nil || choice < 1 || choice > maxChoice {
		clearScreen()
		return 0, fmt.Errorf("无效的选择，请重新输入。")
	}
	return choice, nil
}

// 进入项目目录并打印目录下的文件夹列表
func runCommand(folder string, subDirs []string, remarks []struct {
	Name   string `json:"name"`
	Remark string `json:"remark"`
}) error {
	fmt.Printf("正在启动项目：%s\n", folder)
	// 切换到指定文件夹
	err := os.Chdir(folder)
	if err != nil {
		return err
	}
	// 判断当前目录是否为子目录
	isSubDir := false
	for _, subDir := range subDirs {
		if subDir == folder {
			isSubDir = true
			break
		}
	}

	if isSubDir {
		//获取子目录路径
		dir, err := os.Getwd()
		if err != nil {
			fmt.Println("无法获取子目录路径:", err)
		}

		// 打印目录下的文件夹列表
		folders, err := listFolders(dir, nil)
		if err != nil {
			return err
		}
		if len(folders) == 0 {
			fmt.Println("项目目录下没有任何文件夹。")
			return nil
		}
		clearScreen()
		printFolderList(folders, subDirs, remarks)

		for {
			choice, err := getUserChoice(len(folders))
			if err != nil {
				fmt.Println(err)
				continue
			}
			selectedFolder := folders[choice-1].Name()
			if err := runCommand(selectedFolder, subDirs, remarks); err != nil {
				return fmt.Errorf("无法执行命令: %v", err)
			}
			break
		}
	} else {
		cmd := exec.Command("code", ".")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		// 检测是否为 WEB 项目
		if _, err := os.Stat("package.json"); err == nil {
			fmt.Printf("检测到 %s 为 WEB 项目\n", folder)
			fmt.Println("5秒后启动 web 服务，Ctrl+C 停止")
			time.Sleep(5 * time.Second)
			cmd := exec.Command("npm", "run", "serve")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println("无法启动 web 服务:", err)
			}
		} else if _, err := os.Stat("webman"); err == nil {
			fmt.Printf("检测到 %s 为 webman 项目\n", folder)
			fmt.Println("5秒后启动 webman 服务，Ctrl+C 停止")
			time.Sleep(5 * time.Second)
			cmd := exec.Command("cmd", "/c", "windows.bat")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println("无法启动 webman 服务:", err)
			}
		}
	}

	return nil
}

// 清屏
func clearScreen() {
	// 判断操作系统类型，清屏命令不同
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}
func contains(str string, subDirs []string) bool {
	for _, s := range subDirs {
		if s == str {
			return true
		}
	}
	return false
}
