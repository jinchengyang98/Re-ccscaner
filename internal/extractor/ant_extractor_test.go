package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAntExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	testDir, err := os.MkdirTemp("", "ant-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// 创建测试文件
	buildXML := `<?xml version="1.0" encoding="UTF-8"?>
<project name="test-project" basedir=".">
    <property name="lib.dir" value="lib"/>
    <property file="dependencies.properties"/>
    
    <dependencies>
        <dependency name="junit" version="4.12" type="test" required="true"/>
        <dependency name="log4j" version="2.14.1" type="runtime" required="false"/>
    </dependencies>
    
    <path id="compile.classpath">
        <location>lib/commons-lang3-3.12.0.jar</location>
        <location>lib/guava-30.1-jre.jar</location>
    </path>
    
    <import file="sub/build.xml"/>
</project>`

	subBuildXML := `<?xml version="1.0" encoding="UTF-8"?>
<project name="sub-project">
    <dependencies>
        <dependency name="mockito" version="3.11.2" type="test" required="true"/>
    </dependencies>
</project>`

	depsProperties := `test.dependency=testng:7.4.0
runtime.dependency=slf4j-api:1.7.32`

	// 写入测试文件
	err = os.WriteFile(filepath.Join(testDir, "build.xml"), []byte(buildXML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(testDir, "sub"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(testDir, "sub", "build.xml"), []byte(subBuildXML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(testDir, "dependencies.properties"), []byte(depsProperties), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 创建lib目录和JAR文件
	err = os.MkdirAll(filepath.Join(testDir, "lib"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// 创建空的JAR文件
	for _, jar := range []string{"commons-lang3-3.12.0.jar", "guava-30.1-jre.jar"} {
		_, err = os.Create(filepath.Join(testDir, "lib", jar))
		if err != nil {
			t.Fatal(err)
		}
	}

	// 创建提取器
	logger, _ := zap.NewDevelopment()
	extractor := NewAntExtractor(logger)

	// 执行测试
	deps, err := extractor.Extract(testDir)
	if err != nil {
		t.Fatal(err)
	}

	// 验证结果
	assert.NoError(t, err)
	assert.Len(t, deps, 7) // 2个直接依赖 + 1个导入依赖 + 2个属性依赖 + 2个JAR依赖

	// 验证直接依赖
	found := false
	for _, dep := range deps {
		if dep.Name == "junit" && dep.Version == "4.12" {
			found = true
			assert.Equal(t, "test", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "ant", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Expected dependency junit:4.12 not found")

	// 验证导入的依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "mockito" && dep.Version == "3.11.2" {
			found = true
			assert.Equal(t, "test", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "ant", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Expected dependency mockito:3.11.2 not found")

	// 验证属性文件中的依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "testng" && dep.Version == "7.4.0" {
			found = true
			assert.Equal(t, "jar", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "ant", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Expected dependency testng:7.4.0 not found")

	// 验证JAR依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "commons-lang3" && dep.Version == "3.12.0" {
			found = true
			assert.Equal(t, "jar", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "ant", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Expected dependency commons-lang3:3.12.0 not found")
}

func TestAntExtractor_ParseJarLocation(t *testing.T) {
	tests := []struct {
		name         string
		location     string
		wantName     string
		wantVersion  string
	}{
		{
			name:        "Standard JAR name",
			location:    "lib/commons-lang3-3.12.0.jar",
			wantName:    "commons-lang3",
			wantVersion: "3.12.0",
		},
		{
			name:        "Multiple hyphens",
			location:    "lib/spring-core-test-5.3.9.jar",
			wantName:    "spring-core-test",
			wantVersion: "5.3.9",
		},
		{
			name:        "No version",
			location:    "lib/mylib.jar",
			wantName:    "mylib",
			wantVersion: "",
		},
		{
			name:        "Not a JAR",
			location:    "lib/mylib.txt",
			wantName:    "",
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotVersion := parseJarLocation(tt.location)
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantVersion, gotVersion)
		})
	}
}

func TestAntExtractor_ParseDependencyValue(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		wantName     string
		wantVersion  string
	}{
		{
			name:        "Colon separated",
			value:       "junit:4.12",
			wantName:    "junit",
			wantVersion: "4.12",
		},
		{
			name:        "JAR file",
			value:       "lib/commons-lang3-3.12.0.jar",
			wantName:    "commons-lang3",
			wantVersion: "3.12.0",
		},
		{
			name:        "Simple value",
			value:       "mylib",
			wantName:    "mylib",
			wantVersion: "",
		},
		{
			name:        "Empty value",
			value:       "",
			wantName:    "",
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotVersion := parseDependencyValue(tt.value)
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantVersion, gotVersion)
		})
	}
}

/*
测试说明:

1. 主要测试用例:
- TestAntExtractor_Extract: 测试完整的依赖提取功能
- TestAntExtractor_ParseJarLocation: 测试JAR文件路径解析
- TestAntExtractor_ParseDependencyValue: 测试依赖值解析

2. 测试覆盖:
- 直接声明的依赖
- 导入文件中的依赖
- 属性文件中的依赖
- JAR文件依赖
- 各种格式的依赖声明

3. 测试数据:
- 模拟真实的build.xml文件
- 包含各种依赖声明方式
- 包含子项目导入
- 包含属性文件
- 包含JAR文件

4. 验证内容:
- 依赖解析的正确性
- 依赖属性的完整性
- 错误处理
- 边界情况

5. 运行方式:
go test -v ./internal/extractor -run "TestAntExtractor"
*/ 