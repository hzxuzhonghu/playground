#!/usr/bin/env bash
#!/bin/bash

# 脚本功能：从文件中读取时间段列表，查询每个时间段内指定用户的贡献值

# AUTHOR="foo@bar.com"
# SINCE_DATE="2022-01-01"
function contributions {
  git log --no-merges --since ${SINCE_DATE} --author "${AUTHOR}"  --numstat |\
    grep -v "version" | grep -v "infra/gen-resourcesdocs/cmd/" | grep -v "infra/gen-resourcesdocs/pkg/" | grep -v "api" | grep -v "reference" |\
    grep -v "vendor" |\
    grep -Pv "Date:|insertion|deletion|file|Bin|\.svg|\.drawio|generated|yaml|\.json|html|go\.sum|\.pb\.go|\.pb-c|\=\>" | sort -k3 |\
    grep -P "^\d+\t\d+" |\
    awk 'BEGIN{total=0}{total+=$1+$2}END{print total}'
}

# AUTHOR="foo@bar.com"
# SINCE_DATE="2022-01-01"
# UNTIL_DATE="2023-01-01"
function contributions-period {
  git log --no-merges --since ${SINCE_DATE} --until ${UNTIL_DATE} --author "${AUTHOR}"  --numstat |\
    grep -v "version" | grep -v "infra/gen-resourcesdocs/cmd/" | grep -v "infra/gen-resourcesdocs/pkg/" | grep -v "api" | grep -v "reference" |\
    grep -v "vendor" |\
    grep -Pv "Date:|insertion|deletion|file|Bin|\.svg|\.drawio|generated|yaml|\.json|html|go\.sum|\.pb\.go|\.pb-c|\=\>" | sort -k3 |\
    grep -P "^\d+\t\d+" |\
    awk 'BEGIN{total=0}{total+=$1+$2}END{print total}'
}

# AUTHOR="foo@bar.com"
# SINCE_DATE="2022-01-01"
# UNTIL_DATE="2023-01-01"
function changes-period {
  git log --no-merges --since ${SINCE_DATE} --until ${UNTIL_DATE} --author "${AUTHOR}"  --numstat |\
    grep -v "vendor" |\
    grep -Pv "Date:|insertion|deletion|file|Bin|\.svg|\.drawio|generated|yaml|\.json|html|go\.sum|\.pb\.go|\.pb-c|\=\>" |\
    grep -P "^\d+\t\d+|^commit|^Author"
}

# 检查参数数量
if [ $# -lt 2 ]; then
    echo "用法: $0 <时间段文件> <用户1> <用户列表文件>"
    echo "示例: $0 time_periods.txt users.txt"
    echo "时间文件格式示例:"
    echo "2024-01-01,2024-01-31"
    echo "2024-02-01,2024-02-28"
    exit 1
fi

# 从命令行参数获取时间范围
TIME_PERIODS_FILE="$1"
USERS_FILE="$2"

# 检查时间文件是否存在
if [ ! -f "$TIME_PERIODS_FILE" ]; then
    echo "错误: 时间段文件 $TIME_PERIODS_FILE 不存在"
    exit 1
fi

# 从文件读取用户或者逐行读取
while IFS= read -r AUTHOR; do
    USERS+=("$AUTHOR")
done < $USERS_FILE

# 检查是否提供了用户列表
if [ ${#USERS[@]} -eq 0 ]; then
    echo "错误: 必须指定至少一个用户"
    exit 1
fi

# 检查contributions-period函数是否可用
if ! type contributions-period >/dev/null 2>&1; then
    echo "错误: contributions-period 函数未定义"
    echo "请确保已加载包含该函数的脚本或环境"
    exit 1
fi

# 读取时间段文件并处理每个时间段
while IFS=',' read -r SINCE_DATE UNTIL_DATE || [ -n "$SINCE_DATE" ]; do
    # 跳过空行和注释行(以#开头的行)
    if [[ -z "$SINCE_DATE" || "$SINCE_DATE" == \#* ]]; then
        continue
    fi
    
    # 验证日期格式(简单验证)
    if ! [[ "$SINCE_DATE" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || 
       ! [[ "$UNTIL_DATE" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
        echo "警告: 跳过无效日期格式的行: $SINCE_DATE,$UNTIL_DATE"
        continue
    fi
    
    echo "========================================"
    echo "处理时间段: $SINCE_DATE 至 $UNTIL_DATE"
    echo "========================================"
    
    # 对每个用户查询贡献值
    for AUTHOR in "${USERS[@]}"; do
        echo "查询用户: $AUTHOR 的贡献值"
        
        # 设置环境变量并调用函数
        AUTHOR="$AUTHOR" \
        SINCE_DATE="$SINCE_DATE" \
        UNTIL_DATE="$UNTIL_DATE" \
        contributions-period
        
        echo "----------------------------------------"
    done
    
    echo ""
done < "$TIME_PERIODS_FILE"

echo "所有时间段处理完成"
