package controller

import (
	"encoding/json"
	"storage-management-system/model"
	"storage-management-system/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
)

type StorageController struct {
	helmService       *service.HelmService
	mongoService      *service.MongoService
	rancherApiService *service.RancherApiService
}

func NewStorageController(h *service.HelmService, m *service.MongoService, r *service.RancherApiService) *StorageController {
	return &StorageController{h, m, r}
}

func (s *StorageController) returnJSON(c *gin.Context, code int, parm interface{}) {
	parmJSONStr := "data"
	if code >= 400 {
		parmJSONStr = "error"
	}
	c.JSON(code, gin.H{
		"code":      code,
		"message":   http.StatusText(code),
		parmJSONStr: parm,
	})
}

func (s *StorageController) CreateStorage(c *gin.Context) {
	var storage model.Storage
	if err := c.ShouldBindJSON(&storage); err != nil {
		s.returnJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	storage.ID = bson.NewObjectId()
	releaseName, resource, err := s.helmService.CreateStorage(&storage)
	if err != nil {
		s.returnJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	storage.ReleaseName = releaseName
	if err := json.Unmarshal(resource, &storage.Resources); err != nil {
		panic(err)
	}
	s.rancherApiService.GetWorkloadStatus(&storage)
	s.rancherApiService.GetServiceEndpoint(&storage)
	s.mongoService.InsertStorage(&storage)
	s.returnJSON(c, http.StatusCreated, storage)
}

func (s *StorageController) ListStorage(c *gin.Context) {
	storageList := s.mongoService.ListStorage()
	for _, storage := range *storageList {
		s.rancherApiService.GetWorkloadStatus(&storage)
		s.rancherApiService.GetServiceEndpoint(&storage)
		s.rancherApiService.GetPVCStatus(&storage)
		if err := json.Unmarshal(s.helmService.GetStorage(storage.ReleaseName), &storage.Resources); err != nil {
			panic(err)
		}
		s.mongoService.UpdateStorage(&storage)
	}
	storageList = s.mongoService.ListStorage()
	s.returnJSON(c, http.StatusOK, storageList)
}

func (s *StorageController) GetStorage(c *gin.Context) {
	releaseName := c.Param("releaseName")
	storage := s.mongoService.GetStorage(releaseName)
	s.rancherApiService.GetWorkloadStatus(storage)
	s.rancherApiService.GetServiceEndpoint(storage)
	s.rancherApiService.GetPVCStatus(storage)
	if err := json.Unmarshal(s.helmService.GetStorage(storage.ReleaseName), &storage.Resources); err != nil {
		panic(err)
	}
	s.mongoService.UpdateStorage(storage)
	s.returnJSON(c, http.StatusOK, storage)
}

func (s *StorageController) DeleteStorage(c *gin.Context) {
	releaseName := c.Param("releaseName")
	s.helmService.DeleteStorage(releaseName)
	s.mongoService.DeleteStorage(releaseName)
	s.returnJSON(c, http.StatusOK, nil)
}
