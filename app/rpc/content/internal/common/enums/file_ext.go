package enums

import "fmt"

// FileExtEnum 上传文件扩展名
type FileExtEnum int32

const (
	FileExtUnknown FileExtEnum = 0
	FileExtJpg     FileExtEnum = 1
	FileExtPng     FileExtEnum = 2
	FileExtGif     FileExtEnum = 3
	FileExtMp4     FileExtEnum = 4
	FileExtMp3     FileExtEnum = 5
	FileExtDoc     FileExtEnum = 6
	FileExtDocx    FileExtEnum = 7
	FileExtPdf     FileExtEnum = 8
	FileExtXls     FileExtEnum = 9
)

var fileExtNames = map[FileExtEnum]string{
	FileExtUnknown: "UNKNOWN",
	FileExtJpg:     "JPG",
	FileExtPng:     "PNG",
	FileExtGif:     "GIF",
	FileExtMp4:     "MP4",
	FileExtMp3:     "MP3",
	FileExtDoc:     "DOC",
	FileExtDocx:    "DOCX",
	FileExtPdf:     "PDF",
	FileExtXls:     "XLS",
}

var fileExtMessages = map[FileExtEnum]string{
	FileExtUnknown: "未知",
	FileExtJpg:     "JPG 图片",
	FileExtPng:     "PNG 图片",
	FileExtGif:     "GIF 图片",
	FileExtMp4:     "MP4 视频",
	FileExtMp3:     "MP3 音频",
	FileExtDoc:     "DOC 文档",
	FileExtDocx:    "DOCX 文档",
	FileExtPdf:     "PDF 文档",
	FileExtXls:     "XLS 表格",
}

func (e FileExtEnum) Int32() int32 {
	return int32(e)
}

func (e FileExtEnum) Valid() bool {
	_, ok := fileExtNames[e]
	return ok
}

func (e FileExtEnum) String() string {
	if name, ok := fileExtNames[e]; ok {
		return name
	}
	return fmt.Sprintf("FileExtEnum(%d)", e)
}

func (e FileExtEnum) Message() string {
	if msg, ok := fileExtMessages[e]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", e)
}

// MIME 扩展名对应 Content-Type
func (e FileExtEnum) MIME() string {
	switch e {
	case FileExtJpg:
		return "image/jpeg"
	case FileExtPng:
		return "image/png"
	case FileExtGif:
		return "image/gif"
	case FileExtMp4:
		return "video/mp4"
	case FileExtMp3:
		return "audio/mpeg"
	case FileExtDoc:
		return "application/msword"
	case FileExtDocx:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case FileExtPdf:
		return "application/pdf"
	case FileExtXls:
		return "application/vnd.ms-excel"
	default:
		return ""
	}
}

// Suffix 扩展名后缀
func (e FileExtEnum) Suffix() string {
	switch e {
	case FileExtJpg:
		return ".jpg"
	case FileExtPng:
		return ".png"
	case FileExtGif:
		return ".gif"
	case FileExtMp4:
		return ".mp4"
	case FileExtMp3:
		return ".mp3"
	case FileExtDoc:
		return ".doc"
	case FileExtDocx:
		return ".docx"
	case FileExtPdf:
		return ".pdf"
	case FileExtXls:
		return ".xls"
	default:
		return ""
	}
}
