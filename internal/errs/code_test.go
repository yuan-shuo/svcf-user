package errs

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeMapCompletenessDynamic 真正动态地检查 codeMsg 和 errorHTTPStatus 的完整性
// 通过解析 code.go 源代码文件，获取所有以 "Code" 开头的 const 定义
func TestCodeMapCompletenessDynamic(t *testing.T) {
	// 动态解析 code.go 文件获取所有错误码常量
	codes, err := parseCodeConstants()
	require.NoError(t, err, "解析 code.go 文件失败")
	require.NotEmpty(t, codes, "未能从 code.go 解析到任何错误码常量")

	t.Logf("从 code.go 解析到 %d 个错误码常量: %v", len(codes), codes)

	// 检查每个错误码在 codeMsg 中是否有定义
	var missingInCodeMsg []int
	for _, code := range codes {
		if _, ok := codeMsg[code]; !ok {
			missingInCodeMsg = append(missingInCodeMsg, code)
		}
	}

	// 检查每个错误码在 errorHTTPStatus 中是否有定义
	var missingInHTTPStatus []int
	for _, code := range codes {
		if _, ok := errorHTTPStatus[code]; !ok {
			missingInHTTPStatus = append(missingInHTTPStatus, code)
		}
	}

	// 断言没有缺失
	assert.Empty(t, missingInCodeMsg, "以下错误码在 codeMsg 中缺失: %v", missingInCodeMsg)
	assert.Empty(t, missingInHTTPStatus, "以下错误码在 errorHTTPStatus 中缺失: %v", missingInHTTPStatus)
}

// parseCodeConstants 解析 code.go 文件，获取所有以 "Code" 开头的 int 类型常量值
// 这是真正动态的，不需要手动维护列表
func parseCodeConstants() ([]int, error) {
	// 获取当前文件所在目录
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		// 如果无法获取当前文件路径，尝试使用相对路径
		return parseCodeFile("code.go")
	}

	// code.go 与当前测试文件在同一目录
	codeFilePath := filepath.Join(filepath.Dir(currentFile), "code.go")
	return parseCodeFile(codeFilePath)
}

// parseCodeFile 解析指定的 Go 源文件，获取所有以 "Code" 开头的 int 类型常量
func parseCodeFile(filename string) ([]int, error) {
	// 解析 Go 源文件
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var codes []int
	seen := make(map[int]bool)

	// 遍历 AST，查找 const 定义
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		// 遍历 const 定义中的所有规格（spec）
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// 遍历所有名称和值
			for i, name := range valueSpec.Names {
				// 只处理以 "Code" 开头的常量
				if !strings.HasPrefix(name.Name, "Code") {
					continue
				}

				// 确保有对应的值
				if i >= len(valueSpec.Values) {
					continue
				}

				// 解析常量值
				value := valueSpec.Values[i]
				if basicLit, ok := value.(*ast.BasicLit); ok && basicLit.Kind == token.INT {
					code, err := strconv.Atoi(basicLit.Value)
					if err == nil && !seen[code] {
						seen[code] = true
						codes = append(codes, code)
					}
				}
			}
		}
	}

	return codes, nil
}
