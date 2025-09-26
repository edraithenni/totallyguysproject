--
-- PostgreSQL database dump
--

\restrict wCyOCQUVlafcTSlVOIotqGnxkC7fNz2cx15YYjeDF0kStLMBuVgwPieSPYQnjdR

-- Dumped from database version 17.6
-- Dumped by pg_dump version 17.6

-- Started on 2025-09-26 13:57:58

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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 227 (class 1259 OID 16541)
-- Name: comments; Type: TABLE; Schema: public; Owner: webappuser
--

CREATE TABLE public.comments (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    review_id bigint,
    user_id bigint,
    content text,
    parent_id bigint,
    value bigint
);


ALTER TABLE public.comments OWNER TO webappuser;

--
-- TOC entry 226 (class 1259 OID 16540)
-- Name: comments_id_seq; Type: SEQUENCE; Schema: public; Owner: webappuser
--

CREATE SEQUENCE public.comments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.comments_id_seq OWNER TO webappuser;

--
-- TOC entry 4974 (class 0 OID 0)
-- Dependencies: 226
-- Name: comments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: webappuser
--

ALTER SEQUENCE public.comments_id_seq OWNED BY public.comments.id;


--
-- TOC entry 229 (class 1259 OID 16613)
-- Name: follows; Type: TABLE; Schema: public; Owner: webappuser
--

CREATE TABLE public.follows (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    follower_id bigint,
    followed_id bigint
);


ALTER TABLE public.follows OWNER TO webappuser;

--
-- TOC entry 228 (class 1259 OID 16612)
-- Name: follows_id_seq; Type: SEQUENCE; Schema: public; Owner: webappuser
--

CREATE SEQUENCE public.follows_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.follows_id_seq OWNER TO webappuser;

--
-- TOC entry 4975 (class 0 OID 0)
-- Dependencies: 228
-- Name: follows_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: webappuser
--

ALTER SEQUENCE public.follows_id_seq OWNED BY public.follows.id;


--
-- TOC entry 220 (class 1259 OID 16490)
-- Name: movies; Type: TABLE; Schema: public; Owner: webappuser
--

CREATE TABLE public.movies (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    omdb_id text,
    title text,
    year text,
    plot text,
    poster text,
    avg_rating numeric,
    genre text,
    director text,
    actors text,
    rating text
);


ALTER TABLE public.movies OWNER TO webappuser;

--
-- TOC entry 219 (class 1259 OID 16489)
-- Name: movies_id_seq; Type: SEQUENCE; Schema: public; Owner: webappuser
--

CREATE SEQUENCE public.movies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.movies_id_seq OWNER TO webappuser;

--
-- TOC entry 4976 (class 0 OID 0)
-- Dependencies: 219
-- Name: movies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: webappuser
--

ALTER SEQUENCE public.movies_id_seq OWNED BY public.movies.id;


--
-- TOC entry 223 (class 1259 OID 16514)
-- Name: playlist_movies; Type: TABLE; Schema: public; Owner: webappuser
--

CREATE TABLE public.playlist_movies (
    playlist_id bigint NOT NULL,
    movie_id bigint NOT NULL
);


ALTER TABLE public.playlist_movies OWNER TO webappuser;

--
-- TOC entry 222 (class 1259 OID 16500)
-- Name: playlists; Type: TABLE; Schema: public; Owner: webappuser
--

CREATE TABLE public.playlists (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    cover text,
    owner_id bigint
);


ALTER TABLE public.playlists OWNER TO webappuser;

--
-- TOC entry 221 (class 1259 OID 16499)
-- Name: playlists_id_seq; Type: SEQUENCE; Schema: public; Owner: webappuser
--

CREATE SEQUENCE public.playlists_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.playlists_id_seq OWNER TO webappuser;

--
-- TOC entry 4977 (class 0 OID 0)
-- Dependencies: 221
-- Name: playlists_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: webappuser
--

ALTER SEQUENCE public.playlists_id_seq OWNED BY public.playlists.id;


--
-- TOC entry 225 (class 1259 OID 16530)
-- Name: reviews; Type: TABLE; Schema: public; Owner: webappuser
--

CREATE TABLE public.reviews (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    movie_id bigint,
    user_id bigint,
    content text,
    rating bigint
);


ALTER TABLE public.reviews OWNER TO webappuser;

--
-- TOC entry 224 (class 1259 OID 16529)
-- Name: reviews_id_seq; Type: SEQUENCE; Schema: public; Owner: webappuser
--

CREATE SEQUENCE public.reviews_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.reviews_id_seq OWNER TO webappuser;

--
-- TOC entry 4978 (class 0 OID 0)
-- Dependencies: 224
-- Name: reviews_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: webappuser
--

ALTER SEQUENCE public.reviews_id_seq OWNED BY public.reviews.id;


--
-- TOC entry 218 (class 1259 OID 16479)
-- Name: users; Type: TABLE; Schema: public; Owner: webappuser
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    email text,
    password text,
    role text,
    verified boolean,
    verification_code text,
    avatar text,
    description text
);


ALTER TABLE public.users OWNER TO webappuser;

--
-- TOC entry 217 (class 1259 OID 16478)
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: webappuser
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO webappuser;

--
-- TOC entry 4979 (class 0 OID 0)
-- Dependencies: 217
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: webappuser
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- TOC entry 4790 (class 2604 OID 16544)
-- Name: comments id; Type: DEFAULT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.comments ALTER COLUMN id SET DEFAULT nextval('public.comments_id_seq'::regclass);


--
-- TOC entry 4791 (class 2604 OID 16616)
-- Name: follows id; Type: DEFAULT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.follows ALTER COLUMN id SET DEFAULT nextval('public.follows_id_seq'::regclass);


--
-- TOC entry 4787 (class 2604 OID 16493)
-- Name: movies id; Type: DEFAULT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.movies ALTER COLUMN id SET DEFAULT nextval('public.movies_id_seq'::regclass);


--
-- TOC entry 4788 (class 2604 OID 16503)
-- Name: playlists id; Type: DEFAULT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.playlists ALTER COLUMN id SET DEFAULT nextval('public.playlists_id_seq'::regclass);


--
-- TOC entry 4789 (class 2604 OID 16533)
-- Name: reviews id; Type: DEFAULT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.reviews ALTER COLUMN id SET DEFAULT nextval('public.reviews_id_seq'::regclass);


--
-- TOC entry 4786 (class 2604 OID 16482)
-- Name: users id; Type: DEFAULT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- TOC entry 4810 (class 2606 OID 16548)
-- Name: comments comments_pkey; Type: CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.comments
    ADD CONSTRAINT comments_pkey PRIMARY KEY (id);


--
-- TOC entry 4813 (class 2606 OID 16618)
-- Name: follows follows_pkey; Type: CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT follows_pkey PRIMARY KEY (id);


--
-- TOC entry 4799 (class 2606 OID 16497)
-- Name: movies movies_pkey; Type: CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.movies
    ADD CONSTRAINT movies_pkey PRIMARY KEY (id);


--
-- TOC entry 4804 (class 2606 OID 16518)
-- Name: playlist_movies playlist_movies_pkey; Type: CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.playlist_movies
    ADD CONSTRAINT playlist_movies_pkey PRIMARY KEY (playlist_id, movie_id);


--
-- TOC entry 4802 (class 2606 OID 16507)
-- Name: playlists playlists_pkey; Type: CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.playlists
    ADD CONSTRAINT playlists_pkey PRIMARY KEY (id);


--
-- TOC entry 4808 (class 2606 OID 16537)
-- Name: reviews reviews_pkey; Type: CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT reviews_pkey PRIMARY KEY (id);


--
-- TOC entry 4795 (class 2606 OID 16486)
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- TOC entry 4811 (class 1259 OID 16554)
-- Name: idx_comments_deleted_at; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE INDEX idx_comments_deleted_at ON public.comments USING btree (deleted_at);


--
-- TOC entry 4814 (class 1259 OID 16629)
-- Name: idx_follower_followed; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE UNIQUE INDEX idx_follower_followed ON public.follows USING btree (follower_id, followed_id);


--
-- TOC entry 4815 (class 1259 OID 16630)
-- Name: idx_follows_deleted_at; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE INDEX idx_follows_deleted_at ON public.follows USING btree (deleted_at);


--
-- TOC entry 4796 (class 1259 OID 16498)
-- Name: idx_movies_deleted_at; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE INDEX idx_movies_deleted_at ON public.movies USING btree (deleted_at);


--
-- TOC entry 4797 (class 1259 OID 16601)
-- Name: idx_movies_omdb_id; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE UNIQUE INDEX idx_movies_omdb_id ON public.movies USING btree (omdb_id);


--
-- TOC entry 4800 (class 1259 OID 16513)
-- Name: idx_playlists_deleted_at; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE INDEX idx_playlists_deleted_at ON public.playlists USING btree (deleted_at);


--
-- TOC entry 4805 (class 1259 OID 16539)
-- Name: idx_reviews_deleted_at; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE INDEX idx_reviews_deleted_at ON public.reviews USING btree (deleted_at);


--
-- TOC entry 4806 (class 1259 OID 16538)
-- Name: idx_user_movie; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE UNIQUE INDEX idx_user_movie ON public.reviews USING btree (movie_id, user_id);


--
-- TOC entry 4792 (class 1259 OID 16488)
-- Name: idx_users_deleted_at; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at);


--
-- TOC entry 4793 (class 1259 OID 16487)
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: webappuser
--

CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);


--
-- TOC entry 4820 (class 2606 OID 16607)
-- Name: comments fk_comments_replies; Type: FK CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.comments
    ADD CONSTRAINT fk_comments_replies FOREIGN KEY (parent_id) REFERENCES public.comments(id);


--
-- TOC entry 4817 (class 2606 OID 16524)
-- Name: playlist_movies fk_playlist_movies_movie; Type: FK CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.playlist_movies
    ADD CONSTRAINT fk_playlist_movies_movie FOREIGN KEY (movie_id) REFERENCES public.movies(id);


--
-- TOC entry 4818 (class 2606 OID 16519)
-- Name: playlist_movies fk_playlist_movies_playlist; Type: FK CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.playlist_movies
    ADD CONSTRAINT fk_playlist_movies_playlist FOREIGN KEY (playlist_id) REFERENCES public.playlists(id);


--
-- TOC entry 4821 (class 2606 OID 16549)
-- Name: comments fk_reviews_comments; Type: FK CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.comments
    ADD CONSTRAINT fk_reviews_comments FOREIGN KEY (review_id) REFERENCES public.reviews(id);


--
-- TOC entry 4822 (class 2606 OID 16619)
-- Name: follows fk_users_followers; Type: FK CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT fk_users_followers FOREIGN KEY (followed_id) REFERENCES public.users(id);


--
-- TOC entry 4823 (class 2606 OID 16624)
-- Name: follows fk_users_following; Type: FK CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.follows
    ADD CONSTRAINT fk_users_following FOREIGN KEY (follower_id) REFERENCES public.users(id);


--
-- TOC entry 4816 (class 2606 OID 16508)
-- Name: playlists fk_users_playlists; Type: FK CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.playlists
    ADD CONSTRAINT fk_users_playlists FOREIGN KEY (owner_id) REFERENCES public.users(id);


--
-- TOC entry 4819 (class 2606 OID 16570)
-- Name: reviews fk_users_reviews; Type: FK CONSTRAINT; Schema: public; Owner: webappuser
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT fk_users_reviews FOREIGN KEY (user_id) REFERENCES public.users(id);


-- Completed on 2025-09-26 13:57:59

--
-- PostgreSQL database dump complete
--

\unrestrict wCyOCQUVlafcTSlVOIotqGnxkC7fNz2cx15YYjeDF0kStLMBuVgwPieSPYQnjdR

