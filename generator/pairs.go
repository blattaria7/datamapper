package generator

import (
	"fmt"

	"github.com/underbek/datamapper/models"
	"golang.org/x/exp/maps"
)

func createModelsPair(from, to models.Struct, pkgPath string, functions models.Functions) (result, error) {
	var fields []FieldsPair
	packages := make(models.Packages)

	var conversions []string
	var withError bool
	if from.Type.Pointer && !to.Type.Pointer {
		conversion, err := getPointerCheck(
			"from",
			to.Type.FullName(pkgPath),
			fmt.Sprintf("errors.New(\"%s is nil\")", from.Type.Name),
		)
		if err != nil {
			return result{}, err
		}

		conversions = append(conversions, conversion)
		packages[models.Package{
			Name: "errors",
			Path: "errors",
		}] = struct{}{}

		withError = true
	}

	if from.Type.Pointer && to.Type.Pointer {
		conversion, err := getPointerCheck(
			"from",
			to.Type.FullName(pkgPath),
			"nil",
		)
		if err != nil {
			return result{}, err
		}

		conversions = append(conversions, conversion)
	}

	fromFields := make(map[string]models.Field)
	for _, field := range from.Fields {
		fromFields[field.Tags[0].Value] = field
	}

	for _, toField := range to.Fields {
		fromField, ok := fromFields[toField.Tags[0].Value]
		if !ok {
			//TODO: warning or error politics
			continue
		}

		pair, packs, err := getFieldsPair(fromField, toField, from, to, pkgPath, functions)
		if err != nil {
			return result{}, err
		}

		maps.Copy(packages, packs)
		fields = append(fields, pair)
	}

	conversions = append(conversions, fillConversions(fields)...)

	return result{
		fields:      fields,
		packages:    packages,
		conversions: conversions,
		withError:   withError,
	}, nil
}

func getFieldsPair(from, to models.Field, fromModel, toModel models.Struct, pkgPath string, functions models.Functions,
) (FieldsPair, models.Packages, error) {

	cf, err := getConversionFunction(from.Type, to.Type, from.Name, functions)
	if err != nil {
		return FieldsPair{}, nil, err
	}

	res := FieldsPair{
		FromName:  from.Name,
		FromType:  from.Type.Name,
		ToName:    to.Name,
		ToType:    to.Type.Name,
		WithError: cf.WithError,
	}

	return fillConversionFunction(res, from, to, fromModel, toModel, cf, pkgPath)
}

func getAssigmentBySameTypes(fromFieldFullName string, fromType, toType models.Type) string {
	if fromType.Pointer == toType.Pointer {
		return fromFieldFullName
	}

	if toType.Pointer {
		return fmt.Sprintf("&%s", fromFieldFullName)
	}

	return fmt.Sprintf("*%s", fromFieldFullName)
}

func fillConversionFunction(pair FieldsPair, fromField, toField models.Field, fromModel, toModel models.Struct,
	cf models.ConversionFunction, pkgPath string) (FieldsPair, models.Packages, error) {
	pkgs := make(models.Packages)
	if cf.Package.Path != "" {
		pkgs[cf.Package] = struct{}{}
	}

	cfCall := getConversionFunctionCall(
		cf,
		fromField.Type,
		toField.Type,
		pkgPath,
		fmt.Sprintf("from.%s", fromField.Name),
	)

	refAssignment := fmt.Sprintf("&from%s", fromField.Name)
	valueAssignment := fmt.Sprintf("from%s", fromField.Name)

	if isNeedPointerCheckAndReturnError(fromField.Type, toField.Type, cf) {
		conversion, err := getPointerCheck(
			fmt.Sprintf("from.%s", fromField.Name),
			toModel.Type.FullName(pkgPath),
			getFieldPointerCheckError(
				fromModel.Type.FullName(pkgPath),
				toModel.Type.FullName(pkgPath),
				fromField.Name,
				toField.Name,
			),
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		pkgs[models.Package{
			Name: "errors",
			Path: "errors",
		}] = struct{}{}

		pair.PointerToValue = true
		pair.Conversions = []string{conversion}
	}

	switch getConversionRule(fromField.Type, toField.Type, cf) {
	case NeedOnlyAssigmentRule:
		pair.Assignment = getAssigmentBySameTypes(
			fmt.Sprintf("from.%s", fromField.Name),
			fromField.Type,
			toField.Type,
		)

		return pair, pkgs, nil

	case NeedCallConversionFunctionRule:
		pair.Assignment = cfCall
		return pair, pkgs, nil

	case NeedCallConversionFunctionSeparatelyRule:
		conversion, err := getPointerConversion(
			fmt.Sprintf("from%s", fromField.Name),
			cfCall,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}
		pair.Conversions = append(pair.Conversions, conversion)
		pair.Assignment = refAssignment
		return pair, pkgs, nil

	case PointerPoPointerConversionFunctionsRule:
		errString, err := getConvertError(
			fromModel.Type.Name,
			fromField.Name,
			toModel.Type.Name,
			toField.Name,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		pkgs[models.Package{
			Name: "fmt",
			Path: "fmt",
		}] = struct{}{}

		conversion, err := getPointerToPointerConversion(
			fmt.Sprintf("from%s", fromField.Name),
			fmt.Sprintf("from.%s", fromField.Name),
			toModel.Type.FullName(pkgPath),
			toField.Type.FullName(pkgPath),
			cfCall,
			errString,
			cf.WithError,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		pkgs[toField.Type.Package] = struct{}{}

		// not use pointer check
		pair.Conversions = []string{conversion}
		pair.Assignment = valueAssignment
		return pair, pkgs, nil

	case NeedCallConversionFunctionWithErrorRule:
		errString, err := getConvertError(
			fromModel.Type.Name,
			fromField.Name,
			toModel.Type.Name,
			toField.Name,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		pkgs[models.Package{
			Name: "fmt",
			Path: "fmt",
		}] = struct{}{}

		conversion, err := getErrorConversion(
			fmt.Sprintf("from%s", fromField.Name),
			toModel.Type.FullName(pkgPath),
			cfCall,
			errString,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		pair.Conversions = append(pair.Conversions, conversion)
		pair.Assignment = valueAssignment
		pair.WithError = true
		if toField.Type.Pointer && !cf.ToType.Pointer {
			pair.Assignment = refAssignment
		}
		return pair, pkgs, nil
	case NeedRangeBySlice:
		resPair, resPkgs, err := fillConversionFunctionBySlice(pair, fromField, toField, fromModel, toModel, cf, pkgPath)
		if err != nil {
			return FieldsPair{}, nil, err
		}
		maps.Copy(pkgs, resPkgs)

		resPair.Assignment = valueAssignment

		return resPair, pkgs, nil
	}

	return FieldsPair{}, nil, fmt.Errorf(
		"%w: from field %s to field %s",
		ErrUndefinedConversionRule,
		fromField.Name,
		toField.Name,
	)
}

func fillConversionFunctionBySlice(pair FieldsPair, fromField, toField models.Field, fromModel, toModel models.Struct,
	cf models.ConversionFunction, pkgPath string) (FieldsPair, models.Packages, error) {

	pkgs := make(models.Packages)

	cfCall := getConversionFunctionCall(
		cf,
		fromField.Type.Additional.(models.SliceAdditional).InType,
		toField.Type.Additional.(models.SliceAdditional).InType,
		pkgPath,
		"item",
	)

	rule := getConversionRule(
		fromField.Type.Additional.(models.SliceAdditional).InType,
		toField.Type.Additional.(models.SliceAdditional).InType,
		cf,
	)

	refAssignment := "&res"
	valueAssignment := "res"

	var conversions []string
	var assigment string

	if isNeedPointerCheckAndReturnError(
		fromField.Type.Additional.(models.SliceAdditional).InType,
		toField.Type.Additional.(models.SliceAdditional).InType,
		cf,
	) {
		conversion, err := getPointerCheck(
			"item",
			toModel.Type.FullName(pkgPath),
			getFieldPointerCheckError(
				fromModel.Type.FullName(pkgPath),
				toModel.Type.FullName(pkgPath),
				fromField.Name,
				toField.Name,
			),
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		pkgs[models.Package{
			Name: "errors",
			Path: "errors",
		}] = struct{}{}

		pair.PointerToValue = true
		conversions = []string{conversion}
	}

	switch rule {
	case NeedOnlyAssigmentRule:
		fromFieldFullName := "item"
		if !fromField.Type.Additional.(models.SliceAdditional).InType.Pointer &&
			toField.Type.Additional.(models.SliceAdditional).InType.Pointer {

			// cannot use reference by range element
			conversions = append(conversions, "res := item")
			fromFieldFullName = "res"
		}

		assigment = getAssigmentBySameTypes(
			fromFieldFullName,
			fromField.Type.Additional.(models.SliceAdditional).InType,
			toField.Type.Additional.(models.SliceAdditional).InType,
		)

	case NeedCallConversionFunctionRule:
		assigment = cfCall

	case NeedCallConversionFunctionWithErrorRule:
		errString, err := getConvertError(
			fromModel.Type.Name,
			fromField.Name,
			toModel.Type.Name,
			toField.Name,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		pkgs[models.Package{
			Name: "fmt",
			Path: "fmt",
		}] = struct{}{}

		conversion, err := getErrorConversion(
			"res",
			toModel.Type.FullName(pkgPath),
			cfCall,
			errString,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		conversions = append(conversions, conversion)
		assigment = valueAssignment
		pair.WithError = true
		if toField.Type.Additional.(models.SliceAdditional).InType.Pointer && !cf.ToType.Pointer {
			assigment = refAssignment
		}

	case NeedCallConversionFunctionSeparatelyRule:
		conversion, err := getPointerConversion(
			"res",
			cfCall,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}
		conversions = append(conversions, conversion)
		assigment = refAssignment

	case PointerPoPointerConversionFunctionsRule:
		errString, err := getConvertError(
			fromModel.Type.Name,
			fromField.Name,
			toModel.Type.Name,
			toField.Name,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		pkgs[models.Package{
			Name: "fmt",
			Path: "fmt",
		}] = struct{}{}

		conversion, err := getPointerToPointerConversion(
			"resPtr",
			"item",
			toModel.Type.FullName(pkgPath),
			toField.Type.Additional.(models.SliceAdditional).InType.FullName(pkgPath),
			cfCall,
			errString,
			cf.WithError,
		)
		if err != nil {
			return FieldsPair{}, nil, err
		}

		// not use pointer check
		conversions = []string{conversion}
		assigment = "resPtr"

	default:
		return FieldsPair{}, nil, fmt.Errorf(
			"%w: from field %s to field %s",
			ErrUndefinedConversionRule,
			fromField.Name,
			toField.Name,
		)
	}

	conversion, err := getSliceConversion(
		fromField.Name,
		toField.Type.Additional.(models.SliceAdditional).InType.FullName(pkgPath),
		assigment,
		conversions,
	)
	if err != nil {
		return FieldsPair{}, nil, err
	}

	pair.Conversions = append(pair.Conversions, conversion)
	pkgs[toField.Type.Additional.(models.SliceAdditional).InType.Package] = struct{}{}

	return pair, pkgs, nil
}
