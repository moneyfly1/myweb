<?php

declare(strict_types=1);

namespace App\Command;

use App\Models\Config;
use App\Services\Cron as CronService;
use App\Services\Detect;
use App\Services\NodeCollector\NodeCollectorService;
use Exception;
use Telegram\Bot\Exceptions\TelegramSDKException;
use function json_decode;
use function mktime;
use function time;

final class Cron extends Command
{
    public string $description = <<<EOL
├─=: php xcat Cron - 站点定时任务，每五分钟
EOL;

    /**
     * @throws TelegramSDKException
     * @throws Exception
     */
    public function boot(): void
    {
        ini_set('memory_limit', '-1');

        // Log current hour & minute
        $hour = (int) date('H');
        $minute = (int) date('i');

        $jobs = new CronService();

        // Run new shop related jobs
        $jobs->processPendingOrder();
        $jobs->processTabpOrderActivation();
        $jobs->processBandwidthOrderActivation();
        $jobs->processTimeOrderActivation();
        $jobs->processTopupOrderActivation();

        // Run user related jobs
        $jobs->expirePaidUserAccount();
        $jobs->sendPaidUserUsageLimitNotification();

        // Run node related jobs
        $jobs->updateNodeIp();

        if ($_ENV['enable_detect_offline']) {
            $jobs->detectNodeOffline();
        }

        // Run node collector job
        $this->runNodeCollector();

        // Run daily job
        if ($hour === Config::obtain('daily_job_hour') &&
            $minute === Config::obtain('daily_job_minute') &&
            time() - Config::obtain('last_daily_job_time') > 86399
        ) {
            $jobs->cleanDb();
            $jobs->resetNodeBandwidth();
            $jobs->resetFreeUserBandwidth();
            $jobs->sendDailyTrafficReport();

            if (Config::obtain('enable_detect_inactive_user')) {
                $jobs->detectInactiveUser();
            }

            if (Config::obtain('remove_inactive_user_link_and_invite')) {
                $jobs->removeInactiveUserLinkAndInvite();
            }

            if (Config::obtain('im_bot_group_notify_diary')) {
                $jobs->sendDiaryNotification();
            }

            $jobs->resetTodayBandwidth();

            if (Config::obtain('im_bot_group_notify_daily_job')) {
                $jobs->sendDailyJobNotification();
            }

            (new Config())->where('item', 'last_daily_job_time')->update([
                'value' => mktime(
                    Config::obtain('daily_job_hour'),
                    Config::obtain('daily_job_minute'),
                    0,
                    (int) date('m'),
                    (int) date('d'),
                    (int) date('Y')
                ),
            ]);
        }

        // Daily finance report
        if (Config::obtain('enable_daily_finance_mail')
            && $hour === 0
            && $minute === 0
        ) {
            $jobs->sendDailyFinanceMail();
        }

        // Weekly finance report
        if (Config::obtain('enable_weekly_finance_mail')
            && $hour === 0
            && $minute === 0
            && date('w') === '1'
        ) {
            $jobs->sendWeeklyFinanceMail();
        }

        // Monthly finance report
        if (Config::obtain('enable_monthly_finance_mail')
            && $hour === 0
            && $minute === 0
            && date('d') === '01'
        ) {
            $jobs->sendMonthlyFinanceMail();
        }

        // Detect GFW
        if (Config::obtain('enable_detect_gfw') && $minute === 0
        ) {
            $detect = new Detect();
            $detect->gfw();
        }

        // Detect ban
        if (Config::obtain('enable_detect_ban') && $minute === 0
        ) {
            $detect = new Detect();
            $detect->ban();
        }

        // Run email queue
        $jobs->processEmailQueue();
    }

    /**
     * 执行节点采集任务
     */
    private function runNodeCollector(): void
    {
        try {
            $service = new NodeCollectorService();
            $config = $service->getConfig();

            // 检查是否启用定时采集
            if (! ($config['enable_schedule'] ?? false)) {
                return;
            }

            // 检查是否到了采集时间
            $lastUpdate = Config::obtain('node_collector_last_update');
            $updateInterval = $config['update_interval'] ?? 3600;

            if ($lastUpdate === null || $lastUpdate === '') {
                // 从未采集过，立即执行
                $service->addLog('定时任务：首次采集节点', 'info');
                $result = $service->runUpdateTask();
                $service->addLog($result['message'], $result['success'] ? 'success' : 'error');
                return;
            }

            $lastUpdateTime = (int) $lastUpdate;
            $nextUpdateTime = $lastUpdateTime + $updateInterval;
            $currentTime = time();

            // 检查是否到了采集时间
            if ($currentTime >= $nextUpdateTime) {
                $service->addLog('定时任务：开始采集节点', 'info');
                $result = $service->runUpdateTask();
                $service->addLog($result['message'], $result['success'] ? 'success' : 'error');
            }
        } catch (Exception $e) {
            // 记录错误但不影响其他任务
            error_log('Node collector cron job error: ' . $e->getMessage());
        }
    }
}
