package storage

import "time"

type Device struct {
	DeviceID        string `json:"device_id"`
	Name            string `json:"name"`
	Platform        string `json:"platform"`
	CPUBrand        string `json:"cpu_brand"`
	CPUCores        int    `json:"cpu_cores"`
	TotalMemory     int64  `json:"total_memory"`
	TotalDisk       int64  `json:"total_disk"`
	OSName          string `json:"os_name"`
	OSVersion       string `json:"os_version"`
	Architecture    string `json:"architecture"`
	BootTime        int64  `json:"boot_time"`
	Uptime          int64  `json:"uptime"`
	NetworkUpload   int64  `json:"network_upload"`
	NetworkDownload int64  `json:"network_download"`
	LastSeen        int64  `json:"last_seen"`
	CreatedAt       int64  `json:"created_at"`
}

func (s *Store) UpsertDevice(d *Device) error {
	now := time.Now().Unix()
	if d.CreatedAt == 0 {
		d.CreatedAt = now
	}
	d.LastSeen = now
	_, err := s.DB.Exec(`
		INSERT INTO devices (
			device_id, name, platform, cpu_brand, cpu_cores, total_memory, total_disk,
			os_name, os_version, architecture, boot_time, uptime,
			network_upload, network_download, last_seen, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(device_id) DO UPDATE SET
			name=excluded.name,
			platform=excluded.platform,
			cpu_brand=excluded.cpu_brand,
			cpu_cores=excluded.cpu_cores,
			total_memory=excluded.total_memory,
			total_disk=excluded.total_disk,
			os_name=excluded.os_name,
			os_version=excluded.os_version,
			architecture=excluded.architecture,
			boot_time=excluded.boot_time,
			uptime=excluded.uptime,
			network_upload=excluded.network_upload,
			network_download=excluded.network_download,
			last_seen=excluded.last_seen
	`, d.DeviceID, d.Name, d.Platform, d.CPUBrand, d.CPUCores, d.TotalMemory, d.TotalDisk,
		d.OSName, d.OSVersion, d.Architecture, d.BootTime, d.Uptime,
		d.NetworkUpload, d.NetworkDownload, d.LastSeen, d.CreatedAt)
	return err
}

func (s *Store) TouchDevice(deviceID string) error {
	_, err := s.DB.Exec(`UPDATE devices SET last_seen=? WHERE device_id=?`, time.Now().Unix(), deviceID)
	return err
}

func (s *Store) ListDevices() ([]Device, error) {
	rows, err := s.DB.Query(`
		SELECT device_id, name, platform, cpu_brand, cpu_cores, total_memory, total_disk,
			os_name, os_version, architecture, boot_time, uptime,
			network_upload, network_download, last_seen, created_at
		FROM devices ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.DeviceID, &d.Name, &d.Platform, &d.CPUBrand, &d.CPUCores,
			&d.TotalMemory, &d.TotalDisk, &d.OSName, &d.OSVersion, &d.Architecture,
			&d.BootTime, &d.Uptime, &d.NetworkUpload, &d.NetworkDownload,
			&d.LastSeen, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) GetDevice(id string) (*Device, error) {
	row := s.DB.QueryRow(`
		SELECT device_id, name, platform, cpu_brand, cpu_cores, total_memory, total_disk,
			os_name, os_version, architecture, boot_time, uptime,
			network_upload, network_download, last_seen, created_at
		FROM devices WHERE device_id=?
	`, id)
	var d Device
	if err := row.Scan(&d.DeviceID, &d.Name, &d.Platform, &d.CPUBrand, &d.CPUCores,
		&d.TotalMemory, &d.TotalDisk, &d.OSName, &d.OSVersion, &d.Architecture,
		&d.BootTime, &d.Uptime, &d.NetworkUpload, &d.NetworkDownload,
		&d.LastSeen, &d.CreatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}
