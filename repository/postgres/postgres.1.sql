--
-- PostgreSQL database dump
--

\restrict N1WgrslCt18117ErVTVId9dhd42MMr0FtTIYkAJOGrJck7CDGs3WtvcQllig9wf

-- Dumped from database version 17.7
-- Dumped by pg_dump version 17.7

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: block_delete(uuid); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.block_delete(v_chain_id uuid) RETURNS TABLE(block_id uuid)
    LANGUAGE plpgsql
    AS $$
declare
    v_refer     int;
    v_block_id  uuid;
    v_block_ids uuid[] = array []::uuid[];
begin
    for v_block_id in delete from links where chain_id = v_chain_id returning links.block_id
        loop
            update blocks set refer = blocks.refer - 1 where id = v_block_id returning blocks.refer into v_refer;
            if v_refer = 0 then
                delete from blocks where id = v_block_id and refer = 0;
            else
                v_block_id := null;
            end if;
            v_block_ids := v_block_ids || v_block_id;
        end loop;
    delete from chains where id = v_chain_id;
    return query
        select * from unnest(v_block_ids) as u where u <> null_uuid();
end
$$;


ALTER FUNCTION public.block_delete(v_chain_id uuid) OWNER TO postgres;

--
-- Name: block_insert(uuid, uuid[], bytea[], bigint[]); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.block_insert(v_file_id uuid, v_block_ids uuid[], v_hashes bytea[], sizes bigint[]) RETURNS TABLE(chain_id uuid)
    LANGUAGE plpgsql
    AS $$
declare
    n_chain_id uuid;
    o_chain_id uuid;
    v_block_id uuid;
    i          int not null default 0;
begin
    insert into chains select returning id into n_chain_id;
    foreach v_block_id in array v_block_ids
        loop
            insert into blocks (id, hash, size) values (v_block_id, v_hashes[i + 1], sizes[i + 1]);
            insert into links (chain_id, block_id, ordinal) values (n_chain_id, v_block_id, i);
            i := i + 1;
        end loop;
    select files.chain_id from files where files.id = v_file_id for update into o_chain_id;
    if not found then
        raise exception 'not found';
    end if;
    update files set chain_id = n_chain_id where files.id = v_file_id;
    return query
        select * from unnest(array [n_chain_id, o_chain_id]::uuid[]) as u where u <> null_uuid();
end
$$;


ALTER FUNCTION public.block_insert(v_file_id uuid, v_block_ids uuid[], v_hashes bytea[], sizes bigint[]) OWNER TO postgres;

--
-- Name: block_select(bytea, bigint); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.block_select(v_hash bytea, v_size bigint) RETURNS TABLE(block_id uuid)
    LANGUAGE plpgsql
    AS $$
declare
begin
    return query
        select id from blocks where hash = v_hash and size = v_size order by refer desc, updated, id;
end
$$;


ALTER FUNCTION public.block_select(v_hash bytea, v_size bigint) OWNER TO postgres;

--
-- Name: block_update(uuid, uuid, uuid); Type: PROCEDURE; Schema: public; Owner: postgres
--

CREATE PROCEDURE public.block_update(IN v_chain_id uuid, IN o_block_id uuid, IN n_block_id uuid)
    LANGUAGE plpgsql
    AS $$
declare
begin
    update blocks set refer = refer + 1 where id = n_block_id;
    if not found then
        raise exception 'not found';
    end if;
    update links set block_id = n_block_id where chain_id = v_chain_id and block_id = o_block_id;
    if not found then
        raise exception 'not found';
    end if;
    delete from blocks where id = o_block_id and refer = 1;
end
$$;


ALTER PROCEDURE public.block_update(IN v_chain_id uuid, IN o_block_id uuid, IN n_block_id uuid) OWNER TO postgres;

--
-- Name: file_delete(text); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.file_delete(v_path text) RETURNS TABLE(block_id uuid)
    LANGUAGE plpgsql
    AS $$
declare
    v_chain_id uuid;
begin
    delete from files where path = v_path returning files.chain_id into v_chain_id;
    if not found then
        return;
    end if;
    return query
        select * from block_delete(v_chain_id);
end
$$;


ALTER FUNCTION public.file_delete(v_path text) OWNER TO postgres;

--
-- Name: file_insert(text); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.file_insert(v_path text) RETURNS uuid
    LANGUAGE plpgsql
    AS $$
declare
    v_file_id uuid;
begin
    insert into files (path) values (v_path) on conflict (path) do update set version = files.version + 1 returning id into v_file_id;
    return v_file_id;
end
$$;


ALTER FUNCTION public.file_insert(v_path text) OWNER TO postgres;

--
-- Name: file_select(text); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.file_select(v_path text) RETURNS TABLE(block_id uuid, size bigint, mime text, created timestamp with time zone)
    LANGUAGE plpgsql
    AS $$
declare
begin
    return query
        select links.block_id, blocks.size, chains.mime, chains.created
        from files
                 join chains on chains.id = files.chain_id
                 join links on chains.id = links.chain_id
                 join blocks on blocks.id = links.block_id
        where files.path = v_path 
        order by links.ordinal;
end
$$;


ALTER FUNCTION public.file_select(v_path text) OWNER TO postgres;

--
-- Name: null_uuid(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.null_uuid() RETURNS uuid
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT '00000000-0000-0000-0000-000000000000'::uuid;
$$;


ALTER FUNCTION public.null_uuid() OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: blocks; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.blocks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    hash bytea NOT NULL,
    size bigint NOT NULL,
    refer integer DEFAULT 1 NOT NULL,
    updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.blocks OWNER TO postgres;

--
-- Name: chains; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chains (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mime text DEFAULT ''::text NOT NULL,
    created timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.chains OWNER TO postgres;

--
-- Name: files; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.files (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    path text NOT NULL,
    chain_id uuid,
    version integer DEFAULT 0 NOT NULL
);


ALTER TABLE public.files OWNER TO postgres;

--
-- Name: links; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.links (
    chain_id uuid NOT NULL,
    block_id uuid NOT NULL,
    ordinal integer NOT NULL
);


ALTER TABLE public.links OWNER TO postgres;

--
-- Name: blocks blocks_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_pk PRIMARY KEY (id);


--
-- Name: chains chains_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chains
    ADD CONSTRAINT chains_pk PRIMARY KEY (id);


--
-- Name: files files_path_idx; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_path_idx UNIQUE (path);


--
-- Name: files files_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_pk PRIMARY KEY (id);


--
-- Name: links links_chain_id_ordinal_idx; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT links_chain_id_ordinal_idx UNIQUE (chain_id, ordinal);


--
-- Name: blocks_hash_size_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX blocks_hash_size_idx ON public.blocks USING btree (hash, size);


--
-- Name: files_chain_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX files_chain_id_idx ON public.files USING btree (chain_id);


--
-- Name: files_path_chain_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX files_path_chain_id_idx ON public.files USING btree (path, chain_id);


--
-- Name: links_block_id_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX links_block_id_idx ON public.links USING hash (block_id);


--
-- Name: links_chain_id_block_id_ordinal_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX links_chain_id_block_id_ordinal_idx ON public.links USING btree (chain_id, block_id, ordinal);


--
-- Name: files files_chain_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_chain_id_fk FOREIGN KEY (chain_id) REFERENCES public.chains(id);


--
-- Name: links links_block_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT links_block_id_fk FOREIGN KEY (block_id) REFERENCES public.blocks(id);


--
-- Name: links links_chain_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT links_chain_id_fk FOREIGN KEY (chain_id) REFERENCES public.chains(id);


--
-- PostgreSQL database dump complete
--

\unrestrict N1WgrslCt18117ErVTVId9dhd42MMr0FtTIYkAJOGrJck7CDGs3WtvcQllig9wf

