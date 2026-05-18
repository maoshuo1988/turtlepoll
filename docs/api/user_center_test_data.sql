-- 用户中心帖子列表测试数据
-- 用法：
-- 1. 把下面的 v_user_id 改成你要测试的真实用户 ID
-- 2. 在 PostgreSQL 中执行本脚本
-- 3. 执行后可调用：
--    GET /api/user/center/topics?user=<v_user_id>&page=1&limit=10

BEGIN;

DO $$
DECLARE
    v_user_id bigint := 1;
    v_now bigint := EXTRACT(EPOCH FROM NOW())::bigint;
BEGIN
    INSERT INTO public.t_topic (
        type,
        node_id,
        user_id,
        title,
        content_type,
        content,
        image_list,
        hide_content,
        vote_id,
        recommend,
        recommend_time,
        sticky,
        sticky_time,
        view_count,
        comment_count,
        like_count,
        status,
        last_comment_time,
        last_comment_user_id,
        user_agent,
        ip,
        ip_location,
        create_time,
        extra_data
    ) VALUES
    (
        0,
        1,
        v_user_id,
        '用户中心测试帖子 01',
        'markdown',
        '这是用户中心接口测试帖子 01 的内容。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        12,
        0,
        0,
        0,
        v_now - 300,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 300,
        ''
    ),
    (
        0,
        1,
        v_user_id,
        '用户中心测试帖子 02',
        'markdown',
        '这是用户中心接口测试帖子 02 的内容。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        8,
        0,
        0,
        0,
        v_now - 200,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 200,
        ''
    ),
    (
        0,
        1,
        v_user_id,
        '用户中心测试帖子 03',
        'markdown',
        '这是用户中心接口测试帖子 03 的内容。',
        '',
        '',
        0,
        false,
        0,
        false,
        0,
        5,
        0,
        0,
        0,
        v_now - 100,
        0,
        'sql-script',
        '127.0.0.1',
        'Local',
        v_now - 100,
        ''
    );
END $$;

COMMIT;
