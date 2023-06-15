create table approved_clients
(
    id       mediumint unsigned auto_increment primary key,
    peer_id  varchar(42)          not null,
    archived tinyint(1) default 0 not null
);

create table mod_core
(
    mod_option  varchar(121)      not null primary key,
    mod_setting int(12)           not null
);

create table torrent_group_freeleech
(
    ID             int(10) auto_increment primary key,
    GroupID        int(10)                                 not null,
    Type           enum ('anime', 'music')                 not null,
    DownMultiplier float                   default 1       not null,
    UpMultiplier   float                   default 1       not null,
    constraint GroupID unique (GroupID, Type)
);

create table torrents
(
    ID             int(10) auto_increment primary key,
    GroupID        int(10)                                 not null,
    TorrentType    enum ('anime', 'music')                 not null,
    info_hash      blob                                    not null,
    Leechers       int(6)                  default 0       not null,
    Seeders        int(6)                  default 0       not null,
    last_action    int                     default 0       not null,
    Snatched       int unsigned            default 0       not null,
    DownMultiplier float                   default 1       not null,
    UpMultiplier   float                   default 1       not null,
    Status         int                     default 0       not null,
    constraint InfoHash unique (info_hash (20))
);

create table transfer_history
(
    uid           int     not null,
    fid           int     not null,
    uploaded      bigint  default 0 not null,
    downloaded    bigint  default 0 not null,
    seeding       tinyint default 0 not null,
    seedtime      int(30) default 0 not null,
    activetime    int(30) default 0 not null,
    hnr           tinyint default 0 not null,
    remaining     bigint  default 0 not null,
    active        tinyint default 0 not null,
    starttime     int     default 0 not null,
    last_announce int     default 0 not null,
    snatched      int     default 0 not null,
    snatched_time int     default 0 not null,
    primary key (uid, fid)
);

create table transfer_ips
(
    last_announce int unsigned       default 0 not null,
    starttime     int unsigned       default 0 not null,
    uid           int unsigned       not null,
    fid           int unsigned       not null,
    ip            int unsigned       not null,
    client_id     mediumint unsigned not null,
    uploaded      bigint unsigned    default 0 not null,
    downloaded    bigint unsigned    default 0 not null,
    port          smallint unsigned zerofill default 0 not null,
    primary key (uid, fid, ip, client_id)
);

create table users_main
(
    ID              int unsigned auto_increment primary key,
    Uploaded        bigint unsigned      default 0   not null,
    Downloaded      bigint unsigned      default 0   not null,
    Enabled         enum ('0', '1', '2') default '0' not null,
    torrent_pass    char(32)                         not null,
    rawup           bigint unsigned      default 0   not null,
    rawdl           bigint unsigned      default 0   not null,
    DownMultiplier  float                default 1   not null,
    UpMultiplier    float                default 1   not null,
    DisableDownload tinyint(1)           default 0   not null,
    TrackerHide     tinyint(1)           default 0   not null
);
