CREATE TABLE IF NOT EXISTS task_dependencies (
    parent_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    child_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (parent_id, child_id)
);
