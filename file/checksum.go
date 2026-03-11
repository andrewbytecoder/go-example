package file

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// GenerateChecksum 计算文件的 MD5 值
// 在校验文件是否变动的时候，直接对比文件的 MD5 值， 而不是对比字段的变动
func GenerateChecksum(configFilePath string) (string, error) {

	// 计算指定文件的 MD5 值
	file, err := os.Open(configFilePath)
	if err != nil {
		return "", err
	}
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// 如果指定文件夹下还有其他文件，需要读取进行计算
	//dir := filepath.Dir(configFilePath)

	return hex.EncodeToString(hash.Sum(nil)), nil
}
