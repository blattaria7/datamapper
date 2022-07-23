package generator

import (
	"errors"
	"os"

	"github.com/underbek/datamapper/models"
	"github.com/underbek/datamapper/utils"
)

var (
	ErrParseError = errors.New("parse error")
	ErrNotFound   = errors.New("not found error")
)

type ConvertorType = string
type ImportType = string

type FieldsPair struct {
	FromName       string
	FromType       string
	ToName         string
	ToType         string
	Assignment     string
	Conversions    []string
	WithError      bool
	PointerToValue bool
}

type result struct {
	convertorName string
	pkg           models.Package
	fromName      string
	toName        string
	fromTag       string
	toTag         string
	fields        []FieldsPair
	packages      map[models.Package]struct{}
	conversions   []string
	withError     bool
}

func CreateConvertor(from, to models.Struct, dest string, functions models.Functions) error {
	content, err := generateConvertor(from, to, dest, functions)
	if err != nil {
		return err
	}

	file, err := os.Create(dest) //nolint:gosec
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }()

	_, err = file.Write(content)
	if err != nil {
		return err
	}

	return nil
}

func generateConvertor(from, to models.Struct, dest string, functions models.Functions) ([]byte, error) {
	pkg, err := utils.LoadPackage(dest)
	if err != nil {
		return nil, err
	}

	res, err := createModelsPair(from, to, pkg.PkgPath, functions)
	if err != nil {
		return nil, err
	}

	res.packages[from.Package] = struct{}{}
	res.packages[to.Package] = struct{}{}

	res.convertorName = generateConvertorName(from, to, pkg.PkgPath)
	res.pkg, err = generateModelPackage(pkg)
	if err != nil {
		return nil, err
	}

	res.fromName = getFullStructName(from, pkg.PkgPath)
	res.toName = getFullStructName(to, pkg.PkgPath)

	res.fromTag = from.Fields[0].Tags[0].Name
	res.toTag = to.Fields[0].Tags[0].Name

	res.withError = isReturnError(res.fields)

	return createConvertor(res)
}
