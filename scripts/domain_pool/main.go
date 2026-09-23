// 订阅域名池命令行工具（面板「系统设置 → 订阅域名池」的 SSH 等价物）
//
// 为什么需要：这个工具的典型使用场景就是「官网域名被墙、面板打不开」——
// 那时后台按钮点不了，但 SSH 还能连上，用它在服务器上一条命令完成换域名：
// 建 nginx 站点 → 签证书（带自动续期钩子）→ 重载 nginx → 写面板配置。
//
// 用法（在服务器上、项目根目录执行）：
//
//	go run ./scripts/domain_pool -list                    # 列出域名池 + 体检
//	go run ./scripts/domain_pool -domain sub2.x.com       # 配置该域名并加入池
//	go run ./scripts/domain_pool -domain sub2.x.com -primary   # 同时设为订阅主域名
//	go run ./scripts/domain_pool -remove sub2.x.com       # 从池里移除并移除站点配置
//
// 也可以用已编译好的二进制（install.sh 构建出的 server 不含本工具，需要单独 go build）：
//
//	go build -o /root/domain_pool ./scripts/domain_pool && /root/domain_pool -list
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/services/domainpool"
	"cboard-go/internal/utils"
)

func main() {
	var (
		domainFlag  = flag.String("domain", "", "要一键配置（或修复）的域名")
		removeFlag  = flag.String("remove", "", "从域名池移除该域名（并移除站点配置，证书保留）")
		listFlag    = flag.Bool("list", false, "列出域名池并做体检")
		primaryFlag = flag.Bool("primary", false, "把 -domain 指定的域名设为订阅主域名")
		timeoutFlag = flag.Int("timeout", 300, "命令超时（秒）")
	)
	flag.Parse()

	if _, err := config.LoadConfig(); err != nil {
		fmt.Printf("❌ 配置加载失败: %v\n", err)
		os.Exit(1)
	}
	if err := database.InitDatabase(); err != nil {
		fmt.Printf("❌ 数据库初始化失败: %v\n", err)
		os.Exit(1)
	}
	db := database.GetDB()
	siteDomain := utils.GetDomainFromDB(db)
	m := domainpool.New("", siteDomain)

	// 防呆：必须在面板项目根目录执行（否则会连到 ./cboard.db，可能是空库/别的库，
	// 把域名池配置写错地方）。这里比对数据库里有没有面板配置来判定。
	if siteDomain == "" && utils.SubscriptionBaseURL(nil, db) == "" {
		wd, _ := os.Getwd()
		fmt.Printf("⚠️ 当前数据库里没有任何面板配置（域名等），很可能连错了库：\n")
		fmt.Printf("   工作目录: %s\n", wd)
		fmt.Printf("   请 cd 到面板项目根目录（含 .env 与 cboard.db 的那一层）再执行本工具。\n")
		fmt.Printf("   本次不做任何写入。\n")
		os.Exit(2)
	}

	primary, backups := domainpool.PoolFromConfig(db)

	if *listFlag || (*domainFlag == "" && *removeFlag == "") {
		available, why := m.Available()
		fmt.Printf("订阅主域名: %s\n备用域名: %v\n网站域名: %s\n", primary, backups, siteDomain)
		if available {
			fmt.Println("一键配置: 可用（root + nginx + certbot 均就绪）")
		} else {
			fmt.Printf("一键配置: 不可用 —— %s\n", why)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutFlag)*time.Second)
		defer cancel()
		for _, st := range m.InspectMany(ctx, domainpool.DomainsForInspect(primary, backups, siteDomain), primary, siteDomain) {
			mark := "✅"
			if !st.HTTPSOK || !st.DNSResolved {
				mark = "⚠️"
			}
			fmt.Printf("%s %-32s DNS=%-8v 站点=%-6v %s HTTPS=%-8v %s\n",
				mark, st.Domain, st.DNSResolved, st.VhostExists,
				certDesc(st), st.HTTPSOK, st.Error)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutFlag)*time.Second)
	defer cancel()

	if *removeFlag != "" {
		domain, err := domainpool.ValidateDomain(*removeFlag)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			os.Exit(1)
		}
		if siteDomain != "" && domain == siteDomain {
			fmt.Println("❌ 这是网站域名，不能从订阅域名池移除")
			os.Exit(1)
		}
		newPrimary := primary
		if domain == primary {
			newPrimary = siteDomain
			for _, b := range backups {
				if b != domain {
					newPrimary = b
					break
				}
			}
		}
		rest := make([]string, 0, len(backups))
		for _, b := range backups {
			if b != domain {
				rest = append(rest, b)
			}
		}
		if err := domainpool.WritePool(db, newPrimary, rest); err != nil {
			fmt.Printf("❌ 写面板配置失败: %v\n", err)
			os.Exit(1)
		}
		steps, err := m.RemoveVhost(ctx, domain)
		for _, s := range steps {
			fmt.Printf("  %s %s — %s\n", mark(s.OK), s.Name, s.Detail)
		}
		if err != nil {
			fmt.Printf("⚠️ 站点移除有告警: %v\n", err)
		}
		fmt.Printf("✅ 已移除 %s（订阅主域名现在为 %s）\n", domain, newPrimary)
		return
	}

	domain, err := domainpool.ValidateDomain(*domainFlag)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	st, steps, cfgErr := m.Configure(ctx, domain)
	for _, s := range steps {
		fmt.Printf("  %s %s — %s\n", mark(s.OK), s.Name, s.Detail)
	}
	if cfgErr != nil {
		fmt.Printf("❌ 配置失败: %v\n", cfgErr)
		os.Exit(1)
	}

	if *primaryFlag || primary == "" {
		primary = st.Domain
	}
	if err := domainpool.WritePool(db, primary, append(backups, st.Domain)); err != nil {
		fmt.Printf("❌ 站点已配好，但写面板配置失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ %s 已加入订阅域名池（订阅主域名=%s）\n", st.Domain, primary)
	fmt.Println("   提醒：客户端要自动轮换到新域名，需要把它加进 App 的域名池并发新版本；")
	fmt.Println("         否则可让用户用「设置 → 服务器线路」手动填写，或从用户面板复制备用订阅地址。")
}

func mark(ok bool) string {
	if ok {
		return "✅"
	}
	return "❌"
}

func certDesc(st domainpool.Status) string {
	if !st.CertExists {
		return "证书=无            "
	}
	renew := "无续期配置"
	if st.AutoRenew {
		renew = "自动续期"
	}
	return fmt.Sprintf("证书=%3d天·%s", st.CertDaysLeft, renew)
}
