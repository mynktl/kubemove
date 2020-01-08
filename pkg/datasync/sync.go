package datasync

func (d *DataSync) Sync() error {
	if d.ds.Status.Status == "Synced" {
		return nil
	}

	//TODO
	return nil
}
