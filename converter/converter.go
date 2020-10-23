package converter

import (
	"context"
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/well"
)

const libreOffice = "/usr/bin/soffice"

// Converter represents the interface of PDF converter
type Converter interface {
	Supported(filename string) bool
	Convert(ctx context.Context, filePath string) (convertedPath string, err error)
}

func NewConverter() Converter {
	return converter{}
}

var supportMap map[string]bool

func init() {
	supportMap = make(map[string]bool)

	// The list was taken from https://en.wikipedia.org/wiki/LibreOffice#Supported_file_formats
	supportList := []string{
		"ABW", "ZABW",
		"PMD", "PM3", "PM4", "PM5", "PM6", "P65",
		"CWK",
		"ASE",
		"AGD", "FHD",
		"KTH", "KEY",
		"NUMBERS",
		"PAGES",
		"PDB",
		"DXF",
		"BMP",
		"CSV", "TXT",
		"CDR", "CMX",
		"CGM",
		"DIF",
		"DBF",
		"XML",
		"EPS",
		"EMF",
		"EPUB",
		"FB2",
		"GPL",
		"GNM", "GNUMERIC",
		"GIF",
		"HWP",
		"PLT",
		"HTML", "HTM",
		"JTD", "JTT",
		"JPG", "JPEG",
		"WK1", "WKS", "123", "WK3", "WK4",
		"PCT",
		"MML",
		"MET",
		"XLS", "XLW", "XLT",
		"IQY",
		"DOCX", "XLSX", "PPTX",
		"PXL",
		"PSW",
		"PPT", "PPS", "POT",
		"PUB",
		"RTF",
		"DOC", "DOT",
		"WPS", "WKS", "WDB",
		"WRI",
		"VSD", "VSDX",
		"PGM", "PBM", "PPM",
		"ODT", "FODT", "ODS", "FODS", "ODP", "FODP", "ODG", "FODG", "ODF",
		"ODB",
		"SXW", "STW", "SXC", "STC", "SXI", "STI", "SXD", "STD", "SXM",
		"PCX",
		"PCD",
		"PSD",
		"PDF",
		"PNG",
		"QXP",
		"WB2", "WQ1", "WQ2",
		"SVG",
		"SGV",
		"602",
		"SDC", "VOR",
		"SDA", "SDD", "SDP", "VOR",
		"SXM",
		"SDW", "SGL", "VOR",
		"SGF",
		"RLF",
		"RAS",
		"SVM",
		"SLK",
		"TIF", "TIFF",
		"TGA",
		"UOF", "UOT", "UOS", "UOP",
		"WMF",
		"WPD",
		"WPS",
		"XBM",
		"XPM",
		"ZMF",
	}
	for _, ext := range supportList {
		supportMap[ext] = true
	}
}

type converter struct{}

func (converter) Supported(filename string) bool {
	ext := filepath.Ext(filename)
	if ext == filename {
		// filepath.Ext(".pdf") returns ".pdf"...
		return false
	}
	if len(ext) == 0 {
		return false
	}

	return supportMap[strings.ToUpper(ext)[1:]]
}

func (converter) Convert(ctx context.Context, filePath string) (convertedPath string, err error) {
	dir := filepath.Dir(filePath)
	profDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	err = well.CommandContext(ctx,
		libreOffice,
		"--headless",
		"--convert-to", "pdf",
		"--outdir", dir,
		"-env:UserInstallation=file://"+profDir,
		filePath).Run()
	if err != nil {
		return "", err
	}

	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, fi := range fis {
		if strings.HasSuffix(strings.ToLower(fi.Name()), ".pdf") {
			return filepath.Join(dir, fi.Name()), nil
		}
	}

	return "", errors.New("no PDF is generated")
}
