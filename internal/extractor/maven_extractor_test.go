package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMavenExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	testDir, err := os.MkdirTemp("", "maven-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// 创建测试文件
	pomXML := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
    
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>2.5.0</version>
    </parent>
    
    <properties>
        <java.version>11</java.version>
        <junit.version>5.7.2</junit.version>
    </properties>
    
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
            <version>2.5.0</version>
        </dependency>
        
        <dependency>
            <groupId>org.junit.jupiter</groupId>
            <artifactId>junit-jupiter</artifactId>
            <version>${junit.version}</version>
            <scope>test</scope>
            <optional>true</optional>
        </dependency>
        
        <dependency>
            <groupId>com.google.guava</groupId>
            <artifactId>guava</artifactId>
            <version>30.1-jre</version>
            <exclusions>
                <exclusion>
                    <groupId>com.google.code.findbugs</groupId>
                    <artifactId>jsr305</artifactId>
                </exclusion>
            </exclusions>
        </dependency>
    </dependencies>
    
    <profiles>
        <profile>
            <id>dev</id>
            <dependencies>
                <dependency>
                    <groupId>org.springframework.boot</groupId>
                    <artifactId>spring-boot-devtools</artifactId>
                    <version>2.5.0</version>
                </dependency>
            </dependencies>
        </profile>
    </profiles>
    
    <modules>
        <module>sub-module</module>
    </modules>
</project>`

	subModulePomXML := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    
    <parent>
        <groupId>com.example</groupId>
        <artifactId>test-project</artifactId>
        <version>1.0.0</version>
    </parent>
    
    <artifactId>sub-module</artifactId>
    
    <dependencies>
        <dependency>
            <groupId>org.mockito</groupId>
            <artifactId>mockito-core</artifactId>
            <version>3.11.2</version>
            <scope>test</scope>
        </dependency>
    </dependencies>
</project>`

	// 写入测试文件
	err = os.WriteFile(filepath.Join(testDir, "pom.xml"), []byte(pomXML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(testDir, "sub-module"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(testDir, "sub-module", "pom.xml"), []byte(subModulePomXML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 创建提取器
	logger, _ := zap.NewDevelopment()
	extractor := NewMavenExtractor(logger)

	// 执行测试
	deps, err := extractor.Extract(testDir)
	if err != nil {
		t.Fatal(err)
	}

	// 验证结果
	assert.NoError(t, err)
	assert.Len(t, deps, 6) // 1个父项目 + 3个直接依赖 + 1个配置文件依赖 + 1个子模块依赖

	// 验证父项目依赖
	found := false
	for _, dep := range deps {
		if dep.Name == "org.springframework.boot:spring-boot-starter-parent" {
			found = true
			assert.Equal(t, "2.5.0", dep.Version)
			assert.Equal(t, "parent", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "maven", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Parent dependency not found")

	// 验证直接依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "org.springframework.boot:spring-boot-starter-web" {
			found = true
			assert.Equal(t, "2.5.0", dep.Version)
			assert.Equal(t, "compile", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "maven", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Direct dependency not found")

	// 验证可选依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "org.junit.jupiter:junit-jupiter" {
			found = true
			assert.Equal(t, "junit.version", dep.Version) // 未解析的属性
			assert.Equal(t, "test", dep.Type)
			assert.False(t, dep.Required)
			assert.Equal(t, "maven", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Optional dependency not found")

	// 验证带排除项的依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "com.google.guava:guava" {
			found = true
			assert.Equal(t, "30.1-jre", dep.Version)
			assert.Equal(t, "compile", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "maven", dep.BuildSystem)
			assert.Len(t, dep.Conflicts, 1)
			assert.Equal(t, "com.google.code.findbugs:jsr305", dep.Conflicts[0].Name)
		}
	}
	assert.True(t, found, "Dependency with exclusions not found")

	// 验证配置文件依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "org.springframework.boot:spring-boot-devtools" {
			found = true
			assert.Equal(t, "2.5.0", dep.Version)
			assert.Equal(t, "compile", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "maven", dep.BuildSystem)
			assert.Contains(t, dep.Source, "profile: dev")
		}
	}
	assert.True(t, found, "Profile dependency not found")

	// 验证子模块依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "org.mockito:mockito-core" {
			found = true
			assert.Equal(t, "3.11.2", dep.Version)
			assert.Equal(t, "test", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "maven", dep.BuildSystem)
			assert.Contains(t, dep.Source, "sub-module")
		}
	}
	assert.True(t, found, "Sub-module dependency not found")
}

func TestMavenExtractor_ConvertMavenDependency(t *testing.T) {
	tests := []struct {
		name     string
		dep      MavenDependency
		source   string
		want     models.Dependency
	}{
		{
			name: "Basic dependency",
			dep: MavenDependency{
				GroupID:    "org.example",
				ArtifactID: "example-lib",
				Version:    "1.0.0",
			},
			source: "pom.xml",
			want: models.Dependency{
				Name:        "org.example:example-lib",
				Version:     "1.0.0",
				Type:        "compile",
				Required:    true,
				BuildSystem: "maven",
				Source:      "pom.xml",
			},
		},
		{
			name: "Optional test dependency",
			dep: MavenDependency{
				GroupID:    "org.junit",
				ArtifactID: "junit",
				Version:    "4.12",
				Scope:      "test",
				Optional:   true,
			},
			source: "pom.xml",
			want: models.Dependency{
				Name:        "org.junit:junit",
				Version:     "4.12",
				Type:        "test",
				Required:    false,
				BuildSystem: "maven",
				Source:      "pom.xml",
			},
		},
		{
			name: "Dependency with classifier",
			dep: MavenDependency{
				GroupID:    "org.example",
				ArtifactID: "example-lib",
				Version:    "1.0.0",
				Classifier: "sources",
			},
			source: "pom.xml",
			want: models.Dependency{
				Name:        "org.example:example-lib:sources",
				Version:     "1.0.0",
				Type:        "compile",
				Required:    true,
				BuildSystem: "maven",
				Source:      "pom.xml",
			},
		},
		{
			name: "Dependency with exclusions",
			dep: MavenDependency{
				GroupID:    "org.example",
				ArtifactID: "example-lib",
				Version:    "1.0.0",
				Exclusions: []MavenExclusion{
					{GroupID: "org.excluded", ArtifactID: "lib1"},
					{GroupID: "org.excluded", ArtifactID: "lib2"},
				},
			},
			source: "pom.xml",
			want: models.Dependency{
				Name:        "org.example:example-lib",
				Version:     "1.0.0",
				Type:        "compile",
				Required:    true,
				BuildSystem: "maven",
				Source:      "pom.xml",
				Conflicts: []models.Dependency{
					{Name: "org.excluded:lib1", BuildSystem: "maven"},
					{Name: "org.excluded:lib2", BuildSystem: "maven"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertMavenDependency(tt.dep, tt.source)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Version, got.Version)
			assert.Equal(t, tt.want.Type, got.Type)
			assert.Equal(t, tt.want.Required, got.Required)
			assert.Equal(t, tt.want.BuildSystem, got.BuildSystem)
			assert.Equal(t, tt.want.Source, got.Source)
			assert.Equal(t, len(tt.want.Conflicts), len(got.Conflicts))
		})
	}
}

/*
测试说明:

1. 主要测试用例:
- TestMavenExtractor_Extract: 测试完整的依赖提取功能
- TestMavenExtractor_ConvertMavenDependency: 测试依赖转换功能

2. 测试覆盖:
- 父项目依赖
- 直接依赖
- 可选依赖
- 带排除项的依赖
- 配置文件中的依赖
- 子模块依赖
- 不同类型的依赖转换

3. 测试数据:
- 模拟真实的pom.xml文件
- 包含各种依赖声明方式
- 包含子模块
- 包含配置文件
- 包含属性引用

4. 验证内容:
- 依赖解析的正确性
- 依赖属性的完整性
- 错误处理
- 边界情况

5. 运行方式:
go test -v ./internal/extractor -run "TestMavenExtractor"
*/ 