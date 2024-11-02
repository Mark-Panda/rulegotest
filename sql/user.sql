-- 创建索引
create sequence users_seq increment by 1 minvalue 1 no maxvalue start with 1;

CREATE TABLE "public"."users" (
    "id" bigint NOT NULL DEFAULT nextval('users_seq'::regclass),
    "name" varchar(64) COLLATE "pg_catalog"."default",
    "age" int4,
    "created_at" timestamptz(6) NOT NULL DEFAULT now(),
    "updated_at" timestamptz(6) NOT NULL DEFAULT now(),
    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);


COMMENT ON TABLE "public"."users" IS '用户测试表';

