{include file='admin/header.tpl'}

<div class="page-wrapper">
    <div class="container-xl">
        <div class="page-header d-print-none text-white">
            <div class="row align-items-center">
                <div class="col">
                    <h2 class="page-title">
                        <span class="home-title">节点采集配置</span>
                    </h2>
                    <div class="page-pretitle my-3">
                        <span class="home-subtitle">
                            配置节点采集源，自动从外部 URL 采集节点
                        </span>
                    </div>
                </div>
                <div class="col-auto ms-auto d-print-none">
                    <div class="btn-list">
                        <button id="run-collector" class="btn btn-primary">
                            <i class="icon ti ti-player-play"></i>
                            立即采集
                        </button>
                        <button id="save-config" class="btn btn-success">
                            <i class="icon ti ti-device-floppy"></i>
                            保存配置
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
    <div class="page-body">
        <div class="container-xl">
            <div class="row row-deck row-cards">
                <div class="col-md-8 col-sm-12">
                    <div class="card">
                        <div class="card-header card-header-light">
                            <h3 class="card-title">采集源配置</h3>
                        </div>
                        <div class="card-body">
                            <div class="form-group mb-3">
                                <label class="form-label required">采集源 URL 列表</label>
                                <div id="urls-container">
                                    <!-- URLs will be added here dynamically -->
                                </div>
                                <button type="button" class="btn btn-sm btn-primary mt-2" id="add-url">
                                    <i class="icon ti ti-plus"></i>
                                    添加 URL
                                </button>
                                <div class="form-hint">
                                    每行一个 URL，支持 Base64 编码的订阅链接
                                </div>
                            </div>
                            <div class="form-group mb-3">
                                <label class="form-label">过滤关键词</label>
                                <div id="keywords-container">
                                    <!-- Keywords will be added here dynamically -->
                                </div>
                                <button type="button" class="btn btn-sm btn-primary mt-2" id="add-keyword">
                                    <i class="icon ti ti-plus"></i>
                                    添加关键词
                                </button>
                                <div class="form-hint">
                                    包含这些关键词的节点将被过滤（不采集）
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="col-md-4 col-sm-12">
                    <div class="card">
                        <div class="card-header card-header-light">
                            <h3 class="card-title">采集设置</h3>
                        </div>
                        <div class="card-body">
                            <div class="form-group mb-3">
                                <label class="form-label">更新间隔（秒）</label>
                                <input id="update_interval" type="number" class="form-control" value="3600" min="60">
                                <div class="form-hint">
                                    定时采集的间隔时间，建议 3600 秒（1小时）
                                </div>
                            </div>
                            <div class="form-group mb-3">
                                <span class="col">启用定时采集</span>
                                <span class="col-auto">
                                    <label class="form-check form-check-single form-switch">
                                        <input id="enable_schedule" class="form-check-input" type="checkbox">
                                    </label>
                                </span>
                                <div class="form-hint">
                                    启用后，系统将按照设定的间隔自动采集节点
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="card mt-3">
                        <div class="card-header card-header-light">
                            <h3 class="card-title">采集状态</h3>
                        </div>
                        <div class="card-body">
                            <div class="mb-2">
                                <strong>最后采集时间：</strong>
                                <span id="last-update">-</span>
                            </div>
                            <div class="mb-2">
                                <strong>下次采集时间：</strong>
                                <span id="next-update">-</span>
                            </div>
                            <div class="mb-2">
                                <strong>状态：</strong>
                                <span id="collector-status" class="badge bg-blue">就绪</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <div class="row row-deck row-cards mt-3">
                <div class="col-12">
                    <div class="card">
                        <div class="card-header card-header-light">
                            <h3 class="card-title">采集日志</h3>
                        </div>
                        <div class="card-body">
                            <div class="table-responsive">
                                <table class="table table-vcenter card-table">
                                    <thead>
                                    <tr>
                                        <th>时间</th>
                                        <th>级别</th>
                                        <th>消息</th>
                                    </tr>
                                    </thead>
                                    <tbody id="logs-tbody">
                                    <!-- Logs will be loaded here -->
                                    </tbody>
                                </table>
                            </div>
                            <button type="button" class="btn btn-sm btn-primary mt-2" id="refresh-logs">
                                <i class="icon ti ti-refresh"></i>
                                刷新日志
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>

<script>
    let config = {
        urls: [],
        update_interval: 3600,
        enable_schedule: false,
        filter_keywords: []
    };

    // 加载配置
    function loadConfig() {
        $.ajax({
            url: '/admin/node/collector/config',
            type: 'GET',
            dataType: 'json',
            success: function(data) {
                if (data.ret === 1) {
                    config = data.config;
                    updateUI();
                    if (data.status) {
                        updateStatus(data.status);
                    }
                }
            }
        });
    }

    // 更新 UI
    function updateUI() {
        // 更新 URLs
        $('#urls-container').empty();
        if (config.urls && config.urls.length > 0) {
            config.urls.forEach(function(url, index) {
                addUrlInput(url);
            });
        } else {
            addUrlInput('');
        }

        // 更新关键词
        $('#keywords-container').empty();
        if (config.filter_keywords && config.filter_keywords.length > 0) {
            config.filter_keywords.forEach(function(keyword) {
                addKeywordInput(keyword);
            });
        } else {
            addKeywordInput('');
        }

        // 更新其他设置
        $('#update_interval').val(config.update_interval || 3600);
        $('#enable_schedule').prop('checked', config.enable_schedule || false);
    }

    // 添加 URL 输入框
    function addUrlInput(value = '') {
        const index = $('#urls-container .url-input-group').length;
        const html = `
            <div class="input-group url-input-group mb-2">
                <input type="text" class="form-control url-input" value="${value}" placeholder="https://example.com/subscribe">
                <button type="button" class="btn btn-sm btn-red remove-url" data-index="${index}">
                    <i class="icon ti ti-x"></i>
                </button>
            </div>
        `;
        $('#urls-container').append(html);
    }

    // 添加关键词输入框
    function addKeywordInput(value = '') {
        const index = $('#keywords-container .keyword-input-group').length;
        const html = `
            <div class="input-group keyword-input-group mb-2">
                <input type="text" class="form-control keyword-input" value="${value}" placeholder="关键词">
                <button type="button" class="btn btn-sm btn-red remove-keyword" data-index="${index}">
                    <i class="icon ti ti-x"></i>
                </button>
            </div>
        `;
        $('#keywords-container').append(html);
    }

    // 更新状态
    function updateStatus(status) {
        if (status.last_update) {
            const date = new Date(status.last_update * 1000);
            $('#last-update').text(date.toLocaleString());
        } else {
            $('#last-update').text('从未采集');
        }

        if (status.next_update) {
            const date = new Date(status.next_update * 1000);
            $('#next-update').text(date.toLocaleString());
        } else {
            $('#next-update').text('-');
        }
    }

    // 加载日志
    function loadLogs() {
        $.ajax({
            url: '/admin/node/collector/logs',
            type: 'GET',
            dataType: 'json',
            data: { limit: 50 },
            success: function(data) {
                if (data.ret === 1 && data.logs) {
                    const tbody = $('#logs-tbody');
                    tbody.empty();
                    data.logs.reverse().forEach(function(log) {
                        const date = new Date(log.time * 1000);
                        const levelClass = log.level === 'error' ? 'bg-red' : (log.level === 'success' ? 'bg-green' : 'bg-blue');
                        const row = `
                            <tr>
                                <td>${date.toLocaleString()}</td>
                                <td><span class="badge ${levelClass}">${log.level}</span></td>
                                <td>${log.message}</td>
                            </tr>
                        `;
                        tbody.append(row);
                    });
                }
            }
        });
    }

    // 保存配置
    $('#save-config').click(function() {
        const urls = [];
        $('#urls-container .url-input').each(function() {
            const url = $(this).val().trim();
            if (url) {
                urls.push(url);
            }
        });

        const keywords = [];
        $('#keywords-container .keyword-input').each(function() {
            const keyword = $(this).val().trim();
            if (keyword) {
                keywords.push(keyword);
            }
        });

        const data = {
            urls: urls,
            update_interval: parseInt($('#update_interval').val()) || 3600,
            enable_schedule: $('#enable_schedule').is(':checked'),
            filter_keywords: keywords
        };

        $.ajax({
            url: '/admin/node/collector/config',
            type: 'POST',
            dataType: 'json',
            data: data,
            success: function(data) {
                if (data.ret === 1) {
                    $('#success-message').text(data.msg);
                    $('#success-dialog').modal('show');
                    loadConfig();
                } else {
                    $('#fail-message').text(data.msg);
                    $('#fail-dialog').modal('show');
                }
            }
        });
    });

    // 立即采集
    $('#run-collector').click(function() {
        const btn = $(this);
        btn.prop('disabled', true).html('<i class="icon ti ti-loader"></i> 采集中...');
        
        $.ajax({
            url: '/admin/node/collector/run',
            type: 'POST',
            dataType: 'json',
            success: function(data) {
                if (data.ret === 1) {
                    $('#success-message').text(data.msg);
                    $('#success-dialog').modal('show');
                    loadLogs();
                    loadConfig();
                } else {
                    $('#fail-message').text(data.msg);
                    $('#fail-dialog').modal('show');
                }
            },
            complete: function() {
                btn.prop('disabled', false).html('<i class="icon ti ti-player-play"></i> 立即采集');
            }
        });
    });

    // 添加 URL
    $('#add-url').click(function() {
        addUrlInput();
    });

    // 添加关键词
    $('#add-keyword').click(function() {
        addKeywordInput();
    });

    // 删除 URL
    $(document).on('click', '.remove-url', function() {
        $(this).closest('.url-input-group').remove();
    });

    // 删除关键词
    $(document).on('click', '.remove-keyword', function() {
        $(this).closest('.keyword-input-group').remove();
    });

    // 刷新日志
    $('#refresh-logs').click(function() {
        loadLogs();
    });

    // 初始化
    $(document).ready(function() {
        loadConfig();
        loadLogs();
        
        // 定时刷新状态和日志
        setInterval(function() {
            $.ajax({
                url: '/admin/node/collector/status',
                type: 'GET',
                dataType: 'json',
                success: function(data) {
                    if (data.ret === 1 && data.status) {
                        updateStatus(data.status);
                    }
                }
            });
        }, 30000); // 每30秒刷新一次
    });
</script>

{include file='admin/footer.tpl'}
