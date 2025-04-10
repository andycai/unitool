package stats

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/andycai/goapi/models"
	"github.com/andycai/goapi/modules/system/adminlog"
	"github.com/gofiber/fiber/v2"
)

// @Summary 创建统计记录（含图片）
// @Description 创建新的统计记录，支持图片上传（base64编码）
// @Tags stats
// @Accept json
// @Produce json
// @Param body body models.StatsRecord true "统计记录数据"
// @Success 201 {object} models.StatsRecord
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/stats [post]
func CreateStats(c *fiber.Ctx) error {
	var record models.StatsRecord
	if err := c.BodyParser(&record); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	// 设置创建时间
	record.CreatedAt = time.Now().UnixMilli()

	var existingRecord models.StatsRecord
	if err := app.DB.Where("login_id = ?", record.LoginID).First(&existingRecord).Error; err != nil {
		if err := app.DB.Create(&record).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot create stats record"})
		}
	}

	var info models.StatsInfo
	if err := c.BodyParser(&info); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	// 设置创建时间
	info.CreatedAt = time.Now().UnixMilli()

	tmp, _ := json.Marshal(&info.Process2)
	info.Process = string(tmp)

	// Handle the pic field
	if info.Pic != "" {
		// Decode base64 to image data
		imgData, err := base64.StdEncoding.DecodeString(info.Pic)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid base64 image data"})
		}

		// Create directory if it doesn't exist
		uploadDir := "./uploads/stats"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create upload directory"})
		}

		// Generate a unique filename
		filename := time.Now().Format("20060102150405") + ".jpg"
		filePath := filepath.Join(uploadDir, filename)

		// Save the image file
		if err := os.WriteFile(filePath, imgData, 0644); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save image file"})
		}

		// Update the pic field with the file path
		info.Pic = filepath.Join("/uploads/stats", filename)
	}

	if err := app.DB.Create(&info).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot create stats record"})
	}

	// 记录操作日志
	// adminlog.CreateAdminLog(c, "create", "stats", record.ID, fmt.Sprintf("创建统计记录：%d", record.LoginID))

	return c.Status(fiber.StatusCreated).JSON(record)
}

// @Summary 获取统计列表
// @Description 获取分页统计记录列表
// @Tags stats
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Param search query string false "搜索关键词"
// @Param date query string false "过滤日期 (格式: YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{} "包含统计列表和分页信息"
// @Failure 500 {object} map[string]string
// @Router /api/stats [get]
func listStatsHandler(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 20)
	searchQuery := c.Query("search", "")
	dateStr := c.Query("date", "")

	query := app.DB.Model(&models.StatsRecord{})

	if searchQuery != "" {
		query = query.Where("login_id LIKE ? OR role_name LIKE ?", "%"+searchQuery+"%", "%"+searchQuery+"%")
	}

	if dateStr != "" {
		date, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			nextDay := date.AddDate(0, 0, 1)
			query = query.Where("created_at >= ? AND created_at < ?", date, nextDay)
		}
	}

	var total int64
	query.Count(&total)

	var stats []models.StatsRecord
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Order("created_at desc").Limit(pageSize).Find(&stats).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot fetch stats records"})
	}

	return c.JSON(fiber.Map{
		"stats":    stats,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// @Summary 删除历史统计
// @Description 删除指定日期前的所有统计记录
// @Tags stats
// @Accept json
// @Produce json
// @Param date query string true "删除截止日期 (格式: YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/stats/before [delete]
func deleteStatsBeforeHandler(c *fiber.Ctx) error {
	dateStr := c.Query("date")
	if dateStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": 1, "error": "date query parameter is required"})
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": 2, "error": "invalid date format"})
	}

	// Convert date to milliseconds timestamp
	timestamp := date.UnixNano() / int64(time.Millisecond)

	// Delete from stats_record table
	if err := app.DB.Where("created_at <= ?", timestamp).Delete(&models.StatsRecord{}).Error; err != nil {
		fmt.Println("err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": 3, "error": "cannot delete stats records"})
	}

	// Fetch stats_info records to delete
	var statsInfo []models.StatsInfo
	if err := app.DB.Where("created_at <= ?", timestamp).Find(&statsInfo).Error; err != nil {
		fmt.Println("err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": 4, "error": "cannot fetch stats info records"})
	}

	// Delete images from file system
	for _, info := range statsInfo {
		picPath := filepath.Join("public", info.Pic)
		fmt.Println("picPath", picPath)
		if info.Pic != "" {
			if err := os.Remove(picPath); err != nil {
				fmt.Println("err", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": 5, "error": "cannot delete image file"})
			}
		}
	}

	// Delete from stats_info table
	result := app.DB.Where("created_at <= ?", timestamp).Delete(&models.StatsInfo{})
	if result.Error != nil {
		fmt.Println("err", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": 6, "error": "cannot delete stats info records"})
	}

	// 记录操作日志
	adminlog.WriteLog(c, "delete", "stats", 0, fmt.Sprintf("批量删除%s之前的统计记录，共%d条", dateStr, result.RowsAffected))

	return c.JSON(fiber.Map{"code": 0, "message": "records deleted successfully", "count": result.RowsAffected})
}

// @Summary 获取统计详情
// @Description 根据登录ID获取详细统计信息
// @Tags stats
// @Accept json
// @Produce json
// @Param login_id query string true "用户登录ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/stats/details [get]
func getStatDetailsHandler(c *fiber.Ctx) error {
	loginID := c.Query("login_id")
	if loginID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "login_id is required"})
	}

	var statsRecord models.StatsRecord
	if err := app.DB.Where("login_id = ?", loginID).First(&statsRecord).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "stats record not found"})
	}

	var statsInfo []models.StatsInfo
	if err := app.DB.Where("login_id = ?", loginID).Order("id desc").Limit(1000).Find(&statsInfo).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot fetch stats info"})
	}

	return c.JSON(fiber.Map{
		"statsRecord": statsRecord,
		"statsInfo":   statsInfo,
	})
}

// @Summary 删除单个统计
// @Description 根据ID删除统计记录及其关联数据
// @Tags stats
// @Accept json
// @Produce json
// @Param id path int true "统计记录ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/stats/{id} [delete]
func deleteStatHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"code": 7, "error": "Stat ID is required"})
	}

	var statsRecord models.StatsRecord
	if err := app.DB.First(&statsRecord, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"code": 8, "error": "Stat record not found"})
	}

	// 获取关联的 StatsInfo 记录
	var statsInfoList []models.StatsInfo
	if err := app.DB.Where("login_id = ?", statsRecord.LoginID).Find(&statsInfoList).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"code": 9, "error": "Failed to fetch associated stats info"})
	}

	// 删除关联的图片文件
	for _, info := range statsInfoList {
		if info.Pic != "" {
			picPath := filepath.Join("public", info.Pic)
			if err := os.Remove(picPath); err != nil {
				// 如果删除失败，记录错误但继续执行
				fmt.Printf("Failed to delete image file %s: %v\n", picPath, err)
			}
		}
	}

	// 删除关联的 StatsInfo 记录
	if err := app.DB.Where("login_id = ?", statsRecord.LoginID).Delete(&models.StatsInfo{}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"code": 10, "error": "Failed to delete associated stats info"})
	}

	// 删除 StatsRecord
	if err := app.DB.Delete(&statsRecord).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"code": 11, "error": "Failed to delete stat record"})
	}

	// 记录操作日志
	adminlog.WriteLog(c, "delete", "stats", statsRecord.ID, fmt.Sprintf("删除统计记录：%d", statsRecord.LoginID))

	return c.JSON(fiber.Map{"code": 0, "message": "Stat record, associated info, and image files deleted successfully"})
}
