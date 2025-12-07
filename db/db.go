package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// NewDB crea la conexión a la base de datos y ejecuta la migración
// solo si el archivo no existe aún.
func NewDB(dataSourceName string) (*sql.DB, error) {
	// Verificar si ya existe el archivo de base de datos
	dbExists := fileExists(dataSourceName)

	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error abriendo la base de datos: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error conectando a la base de datos: %w", err)
	}

	// Solo ejecuta la migración si el archivo no existía
	if !dbExists {
		log.Println("Base de datos no encontrada. Ejecutando migración inicial...")
		if err := migrate(db); err != nil {
			return nil, fmt.Errorf("error en migración: %w", err)
		}
		log.Println("Migración completada exitosamente")
	} else {
		log.Println("Base de datos existente detectada. No se ejecuta migración.")
	}

	log.Println("Conexión a SQLite establecida.")
	return db, nil
}

// fileExists verifica si un archivo existe en el sistema
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// migrate crea las tablas iniciales y el usuario admin
func migrate(db *sql.DB) error {
	schema := `
-- =============================================
-- TABLA DE USUARIOS
-- =============================================

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    ci TEXT DEFAULT '0',
    lastname TEXT NOT NULL,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role INTEGER DEFAULT 5,

    FOREIGN KEY (role) REFERENCES roles(id) 
);

-- =============================================
-- TABLA DE ROLES
-- =============================================

CREATE TABLE IF NOT EXISTS roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    role TEXT NOT NULl
);

-- =============================================
-- TABLA DE BEARER TOKENS
-- =============================================

CREATE TABLE IF NOT EXISTS bearer_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token TEXT NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    created_at TEXT,
    expires_at TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- =============================================
-- TABLA DE PROYECTOS AGRÍCOLAS
-- =============================================
CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    descripcion TEXT NOT NULL,
    fecha_inicio TEXT NOT NULL,
    fecha_cierre TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT 'Habilitado',
    created_at TEXT
);

-- =============================================
-- TABLA DE LABORES AGRONOMICAS
-- =============================================
CREATE TABLE IF NOT EXISTS farm_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    descripcion TEXT NOT NULL
);

-- =============================================
-- TABLA DE EQUIPOS E IMPLEMENTOS
-- =============================================
CREATE TABLE IF NOT EXISTS tools (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    descripcion TEXT NOT NULL
);

-- =============================================
-- TABLA DE DATOS DE PROYECTO
-- =============================================
CREATE TABLE IF NOT EXISTS projects_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    activity TEXT NOT NULL,
    fk_farm_task INTEGER,
    fk_project INTEGER,
    fk_user INTEGER,
    num_human_resources INTEGER,
    cost MONEY,
    details TEXT NOT NULL DEFAULT 'Ninguna',

    FOREIGN KEY (fk_farm_task) REFERENCES farm_tasks(id),
    FOREIGN KEY (fk_project) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (fk_user) REFERENCES users(id)
);

-- =============================================
-- TABLA DE EVENTOS/LOGGER
-- =============================================

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event TEXT NOT NULL,
    module TEXT,
    created_at TEXT,
    timestamp TEXT,
    fk_user INTEGER,

    FOREIGN KEY (fk_user) REFERENCES users(id) ON DELETE CASCADE
);
-- =============================================
-- TABLA DE UNIDADES ESTANDARIZADAS
-- =============================================
CREATE TABLE IF NOT EXISTS units (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    unit TEXT NOT NULL
);
-- =============================================
-- TABLA DE UNIDADES DE MEDIDAS
-- =============================================
CREATE TABLE IF NOT EXISTS measurements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    dimension TEXT NOT NULL,
    fk_unit INTEGER,

    FOREIGN KEY (fk_unit) REFERENCES units(id) ON DELETE CASCADE
);
-- =============================================
-- TABLA INTERMEDIA DE DATOS DE PROYECTO - EQUIPOS E IMPLEMENTOS
-- =============================================
CREATE TABLE IF NOT EXISTS projects_data_tools (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fk_projects_data INTEGER,
    fk_tools INTEGER,

    FOREIGN KEY (fk_tools) REFERENCES tools(id),
    FOREIGN KEY (fk_projects_data) REFERENCES projects_data(id) ON DELETE CASCADE
);


-- =============================================
-- RELACIÓN USUARIOS - PROYECTOS
-- =============================================
CREATE TABLE IF NOT EXISTS user_projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    project_id INTEGER NOT NULL,
    role_in_project TEXT DEFAULT 'colaborador',
    assigned_at TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE(user_id, project_id)
);

-- =============================================
-- TABLA DE PLANES DE ACCIÓN
-- =============================================
CREATE TABLE IF NOT EXISTS action_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    id_project INTEGER,
    id_farm_task INTEGER,
    action_description TEXT,
    fecha_inicio TEXT,
    fecha_cierre TEXT, 
    cantidad_horas FLOAT,
    id_responsable INTEGER,
    tiempo_recurso_humano FLOAT,
    cantidad_de_dias TEXT,
    costo_recurso_humano FLOAT,
    monto_recurso_humano FLOAT GENERATED ALWAYS AS (costo_recurso_humano * tiempo_recurso_humano),
    categoria_insumo_material TEXT,
    descripcion_insumo_material TEXT,
    cantidad_insumo_material INTEGER,
    costo_insumo_material FLOAT,
    id_medida INTEGER, 
    monto_insumo_material FLOAT GENERATED ALWAYS AS (costo_insumo_material * cantidad_insumo_material),
    monto_total FLOAT GENERATED ALWAYS AS (monto_recurso_humano + monto_insumo_material),

    FOREIGN KEY (id_project) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (id_farm_task) REFERENCES farm_tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (id_responsable) REFERENCES users(id) ON DELETE CASCADE
);

-- =============================================
-- USUARIO ADMINISTRADOR INICIAL Y TRABAJADORES INICIALES
-- =============================================
INSERT INTO users (username, name, lastname, email, password_hash, role, ci)
VALUES ('root', 'root', 'root','root@example.com', '$2a$10$.e2jTOtVHftDwmE5N2ig2eCvkMKzF3Y8UZu3Qg9t4NwzwLUlrh.Ou', 1, '0'),
('arcargotte', 'Alan', 'Argotte','alan@example.com', '$2a$10$.e2jTOtVHftDwmE5N2ig2eCvkMKzF3Y8UZu3Qg9t4NwzwLUlrh.Ou', 1, 'V-29765711'), 
('yisus', 'Jesus', 'Gutierrez','jesus@example.com', '$2a$10$.e2jTOtVHftDwmE5N2ig2eCvkMKzF3Y8UZu3Qg9t4NwzwLUlrh.Ou', 1, 'V-29765712');

-- =============================================
-- ROLES INICIALES
-- =============================================
INSERT INTO roles (role)
VALUES ("Administrador"), ("Gerente"), ("Analista"), ("Vendedor"), ("Colaborador"), ("Encargado");

-- =============================================
-- PROYECTOS INICIALES
-- =============================================
INSERT INTO projects (descripcion, fecha_inicio, fecha_cierre, created_at)
VALUES ("Proy 1", "2025-01-01", "2025-01-01", "2025-1-1"),
       ("Proy 2", "2025-02-01", "2025-02-01", "2025-2-1"),
       ("Proy 3", "2025-03-01", "2025-03-01", "2025-3-1");

-- =============================================
-- EQUIPOS E IMPLEMENTOS INICIALES
-- =============================================
INSERT INTO tools (descripcion)
VALUES ("Hacha"), ("Desmalezadora"), ("Machete"), ("Motosierra");

-- =============================================
-- UNIDADES ESTANDARIZADAS INICIALES
-- =============================================
INSERT INTO units (unit)
VALUES ("Pulgadas"), ("Galon");

-- =============================================
-- LABORES AGRONÓMICAS INICIALES
-- =============================================
INSERT INTO farm_tasks (descripcion)
VALUES ("Siembra"), ("Preparación del Suelo"), ("Riego"), ("Control de Plagas y Enfermedades"), ("Cosecha");
	`
	_, err := db.Exec(schema)
	db.Exec("PRAGMA foreign_keys = ON;")

	return err
}
